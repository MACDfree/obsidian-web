可以打开本地obsidian的地址

https://help.obsidian.md/Extending+Obsidian/Obsidian+URI

sudo docker build -t obsidian-web:v4 .

sudo docker run -it --rm -v ~/.ssh/privatekey:/root/.ssh/privatekey -v ./note:/app/note -p 8989:8888 --entrypoint git obsidian-web:v4 clone git@github.com:MACDfree/obnote.git

sudo docker run -it --rm -v ./templates:/app/templates -v ./static:/app/static -v ./config.yml:/app/config.yml -v ./note:/app/note -p 8989:8888 obsidian-web:v4
