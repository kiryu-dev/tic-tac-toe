remove-containers:
	docker rm stateful-server-1
	docker rm stateful-server-2
	docker rm stateful-server-3

remove-images:
	docker rmi tic-tac-toe-server-1
	docker rmi tic-tac-toe-server-2
	docker rmi tic-tac-toe-server-3

rebuild-compose: remove-containers remove-images
	docker compose up