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

#### Holdings

- Login to ETrade.
- Hover over "At Work" in the top navbar and select "Holdings".
- Click on "View by Status".
- Click on "Download" and select "Download Expanded".
- This will download the ETrade Holdings file in XLSX format.

#### Gains and Losses

- Login to ETrade.
- Hover over "At Work" in the top navbar and select "My Account".
- Select "Gains and Losses".
- Click on "Tax Year" and select "Custom Date".
- For "Date Range", select the Indian Fianancial year. e.g. 1 January 2023 to 31 March 2024.
    - The records for 1 January to 31 December(end of previous year in Indian financial year) are required for Schedule FA and the records from 1 April(start of current Indian financial year year) to 31 March(end of Indian financial year year) are required for Schedule CG and Schedule AL.
    - Note: The date values are in `MM/DD/YYYY` format.
    - Click on the calendar icon of each date to ensure you have selected the correct value.
- Ensure all other dropdowns are selected as "All".
- Ensure `Wash sale adjustment` is selected.
- Click on "Apply".
- Click on "Download" and select "Download Expanded".

### Generate data for ITR

```
poetry run generate-itr-data --financial-year "2023-2024" --sbi-reference-rates "./data/SBI_REFERENCE_RATES_USD.csv" --etrade-holdings "<path_to_file>" --etrade-sale-transactions "<path_to_file>"
```
