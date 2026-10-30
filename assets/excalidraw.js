// Excalidraw 只读预览；画布数据只在本页面解压和渲染。
let rendererPromise;
let decompressorPromise;

function loadRenderer() {
    if (!rendererPromise) {
        rendererPromise = import('https://esm.sh/@excalidraw/excalidraw@0.18.0?bundle')
            .catch(error => { rendererPromise = undefined; throw error; });
    }
    return rendererPromise;
}

function loadDecompressor() {
    if (!decompressorPromise) {
        decompressorPromise = import('https://esm.sh/lz-string@1.5.0')
            .catch(error => { decompressorPromise = undefined; throw error; });
    }
    return decompressorPromise;
}

// 在嵌入请求进行时预加载，失败时渲染阶段可以再次尝试。
export function preloadExcalidraw() {
    void loadRenderer().catch(() => {});
    void loadDecompressor().catch(() => {});
}

export async function renderExcalidraw(root) {
    const viewers = Array.from(root.querySelectorAll('.excalidraw-viewer:not([data-rendering])'));
    viewers.forEach(viewer => { viewer.dataset.rendering = 'true'; });
    if (!viewers.length) return;
    const renderer = loadRenderer();
    void renderer.catch(() => {});
    // 同时启动解压库加载，不让画布依次等待网络。
    const decompressor = viewers.some(viewer => viewer.dataset.encoding === 'compressed-json')
        ? loadDecompressor() : null;
    if (decompressor) void decompressor.catch(() => {});
    await Promise.all(viewers.map(async viewer => {
        try {
            const data = Uint8Array.from(atob(viewer.dataset.drawing), c => c.charCodeAt(0));
            let json = new TextDecoder().decode(data);
            if (viewer.dataset.encoding === 'compressed-json') {
                const { decompressFromBase64 } = await decompressor;
                json = decompressFromBase64(json.replace(/\s/g, ''));
            }
            const { exportToSvg } = await renderer;
            const scene = JSON.parse(json);
            if (!Array.isArray(scene.elements)) throw new Error('画布数据无效');
            const elements = scene.elements.filter(element => !element.isDeleted);
            if (!elements.length) { viewer.textContent = '画布为空'; return; }
            // 仅使用画布中内嵌的数据，避免将私有笔记中的地址交给第三方。
            const files = Object.fromEntries(Object.entries(scene.files || {}).filter(([, file]) => typeof file.dataURL === 'string' && /^data:image\//i.test(file.dataURL)));
            const svg = await exportToSvg({ elements, appState: { ...scene.appState, exportWithDarkMode: false }, files });
            const viewport = document.createElement('div');
            viewport.className = 'excalidraw-viewport';
            const controls = document.createElement('div');
            controls.className = 'excalidraw-controls';
            const caption = document.createElement('span');
            caption.className = 'excalidraw-caption';
            caption.textContent = '画布';
            controls.appendChild(caption);
            let zoom = 1;
            const zoomLabel = document.createElement('span');
            zoomLabel.className = 'excalidraw-zoom';
            zoomLabel.textContent = '100%';
            const setZoom = value => {
                zoom = Math.min(4, Math.max(0.25, value));
                svg.style.width = `max(${zoom * 100}%, ${zoom * 720}px)`;
                zoomLabel.textContent = `${Math.round(zoom * 100)}%`;
            };
            // 围绕手势位置缩放，并保持该位置对应的画布内容不变。
            const zoomAt = (value, clientX, clientY) => {
                const rect = viewport.getBoundingClientRect();
                const x = Number.isFinite(clientX) ? clientX - rect.left : viewport.clientWidth / 2;
                const y = Number.isFinite(clientY) ? clientY - rect.top : viewport.clientHeight / 2;
                const oldWidth = svg.getBoundingClientRect().width;
                if (!oldWidth) return;
                const scrollX = viewport.scrollLeft;
                const scrollY = viewport.scrollTop;
                setZoom(value);
                const ratio = svg.getBoundingClientRect().width / oldWidth;
                viewport.scrollLeft = (scrollX + x) * ratio - x;
                viewport.scrollTop = (scrollY + y) * ratio - y;
            };
            let gestureActive = false;
            let gestureStartZoom = 1;
            // Chrome、Edge 的触摸板捏合表现为带 Ctrl 的 wheel。
            viewport.addEventListener('wheel', event => {
                if (!event.ctrlKey || !event.cancelable) return;
                event.preventDefault();
                if (gestureActive) return;
                const delta = event.deltaY * (event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? viewport.clientHeight : 1);
                zoomAt(zoom * Math.exp(-delta * 0.01), event.clientX, event.clientY);
            }, { passive: false });
            // Safari 使用 gesture 事件，scale 相对于手势开始时的大小。
            viewport.addEventListener('gesturestart', event => {
                event.preventDefault();
                gestureActive = true;
                gestureStartZoom = zoom;
            }, { passive: false });
            viewport.addEventListener('gesturechange', event => {
                event.preventDefault();
                if (gestureActive && Number.isFinite(event.scale)) {
                    zoomAt(gestureStartZoom * event.scale, event.clientX, event.clientY);
                }
            }, { passive: false });
            viewport.addEventListener('gestureend', event => {
                event.preventDefault();
                gestureActive = false;
            }, { passive: false });
            for (const [symbol, label, factor] of [['−', '缩小画布', 0.8], ['+', '放大画布', 1.25], ['适应', '重置画布缩放', 0]]) {
                const button = document.createElement('button');
                button.type = 'button'; button.textContent = symbol;
                button.setAttribute('aria-label', label); button.title = label;
                button.addEventListener('click', () => setZoom(factor ? zoom * factor : 1));
                if (factor === 0) {
                    button.classList.add('excalidraw-text-button');
                    controls.appendChild(zoomLabel);
                }
                controls.appendChild(button);
            }
            const previewButton = document.createElement('button');
            previewButton.type = 'button';
            previewButton.classList.add('excalidraw-text-button');
            let previousOverflow = '';
            const updatePreviewButton = () => {
                const active = viewer.classList.contains('is-expanded');
                previewButton.textContent = active ? '关闭预览' : '放大查看';
                previewButton.setAttribute('aria-label', active ? '关闭画布预览' : '放大查看画布');
                previewButton.title = active ? '关闭画布预览' : '放大查看画布';
                previewButton.setAttribute('aria-pressed', String(active));
            };
            const closePreview = () => {
                if (!viewer.classList.contains('is-expanded')) return;
                viewer.classList.remove('is-expanded');
                document.body.style.overflow = previousOverflow;
                updatePreviewButton();
                previewButton.focus();
            };
            previewButton.addEventListener('click', () => {
                if (viewer.classList.contains('is-expanded')) { closePreview(); return; }
                previousOverflow = document.body.style.overflow;
                document.body.style.overflow = 'hidden';
                viewer.classList.add('is-expanded');
                updatePreviewButton();
            });
            document.addEventListener('keydown', event => {
                if (event.key === 'Escape') closePreview();
            });
            updatePreviewButton();
            controls.appendChild(previewButton);
            setZoom(1); svg.style.height = 'auto';
            viewport.appendChild(svg); viewer.replaceChildren(controls, viewport);
            viewer.removeAttribute('data-drawing');
        } catch (error) {
            viewer.textContent = '画布数据无法显示。';
            console.error('Excalidraw 渲染失败', error);
        }
    }));
}
