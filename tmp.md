可以打开本地obsidian的地址

https://help.obsidian.md/Extending+Obsidian/Obsidian+URI

sudo docker build -t obsidian-web:v4 .

sudo docker run -it --rm -v ~/.ssh/privatekey:/root/.ssh/privatekey -v ./note:/app/note -p 8989:8888 --entrypoint git obsidian-web:v4 clone git@github.com:MACDfree/obnote.git

sudo docker run -it --rm -v ~/.ssh/id_ed25519:/root/.ssh/id_ed25519 -v ./note:/app/note --entrypoint git obsidian-web:1.0.0 clone git@github.com:MACDfree/obnote.git note

sudo docker run -it --rm -v ./templates:/app/templates -v ./static:/app/static -v ./config.yml:/app/config.yml -v ./note:/app/note -p 8989:8888 obsidian-web:v4

sudo docker compose run --rm --entrypoint sh obsidian-web -c "chown root:root /root/.ssh/id_ed25519 && chmod 600 /root/.ssh/id_ed25519 && git clone git@github.com:MACDfree/obnote.git note"