# itr-foreign-assets-helper

This project is a helper for generating data related to ITR foreign assets.

## Supported brokers

- ETrade

## Usage

### Install dependencies

Install [poetry](https://github.com/python-poetry/poetry).

```
make poetry-install
```

### Start poetry shell

```
make poetry-shell
```

### Fetch data from broker

#### ETrade

- Login to ETrade.
- Hover over "At Work" in the top navbar and select "Holdings".
- Click on "View by Status".
- Click on "Download" and select "Download Expanded".
- This will download the ETrade Holdings file in XLSX format.

### Generate data for ITR

```
poetry run generate-itr-data --financial-year "2023-2024" --sbi-reference-rates "./data/SBI_REFERENCE_RATES_USD.csv" --etrade-holdings "<path_to_file>"
```
