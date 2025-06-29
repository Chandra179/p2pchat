.PHONY: push peer relay

push:
	git add .
	git commit -m "add"
	git push

rebase:
	git pull --rebase

peer:
	go run main.go --mode=peer

relay:
	go run main.go --mode=relay