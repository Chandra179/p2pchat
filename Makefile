.PHONY: push

push:
	git add .
	git commit -m "add"
	git push

rebase:
	git pull --rebase