.PHONY: build clean

NAME ?= protoc-gen-d2
OUT ?= testdata/generated

all: build $(OUT)

build: $(NAME)

$(NAME):
	go build -o $(NAME)

$(OUT): $(NAME)
	rm -rf $(OUT) 2>/dev/null
	mkdir -p $(OUT)
	protoc \
		-I ./testdata/proto \
		--plugin=$(NAME)=./$(NAME) \
		--d2_out="$(OUT)" \
		./testdata/proto/*.proto

clean:
	rm -rf $(NAME) $(OUT)
