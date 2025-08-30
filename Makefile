.PHONY: build test run clean

build:
	go build -o tmp/itr-foreign-assets-helper cmd/main.go

test:
	go test -v ./...

run: build
	./tmp \
		--financial-year "2023-2024" \
		--etrade-holdings "data/holdings.xlsx" \
		--etrade-sale-transactions "data/gains_losses.xlsx" \
		--sbi-reference-rates "data/SBI_REFERENCE_RATES_USD.csv"

clean:
	rm -rf tmp/ output/

install-deps:
	go mod download
	go mod tidy

fmt:
	go fmt ./...

lint:
	golangci-lint run

# Development helpers
download-sbi-rates:
	curl -o data/SBI_REFERENCE_RATES_USD.csv \
		https://raw.githubusercontent.com/sahilgupta/sbi-fx-ratekeeper/main/csv_files/SBI_REFERENCE_RATES_USD.csv