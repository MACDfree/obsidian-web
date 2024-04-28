可以打开本地obsidian的地址

https://help.obsidian.md/Extending+Obsidian/Obsidian+URI

sudo docker run -it --rm -v ./templates:/app/templates -v ./config.yml:/app/config.yml -v /pathtonote:/note -p 8989:8888 obsidian-web:v4
