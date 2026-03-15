IMAGE := handy-cap-bot
VOLUME := bot-data

.PHONY: build run stop restart logs test clean

build:
	docker build -t $(IMAGE) .

run: build
	docker run -d --restart=unless-stopped \
		--name $(IMAGE) \
		-v $(VOLUME):/data \
		--env-file .env \
		$(IMAGE)

stop:
	docker stop $(IMAGE) && docker rm $(IMAGE)

restart: build
	-docker stop $(IMAGE) 2>/dev/null
	-docker rm $(IMAGE) 2>/dev/null
	docker run -d --restart=unless-stopped \
		--name $(IMAGE) \
		-v $(VOLUME):/data \
		--env-file .env \
		$(IMAGE)

logs:
	docker logs -f $(IMAGE)

test:
	go test -race ./...

clean:
	docker rmi $(IMAGE) 2>/dev/null; docker volume rm $(VOLUME) 2>/dev/null; true
