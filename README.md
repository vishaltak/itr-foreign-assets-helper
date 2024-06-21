# itr-foreign-assets-helper

This project is a helper for generating data related to ITR foreign assets.

## Supported platforms

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

### Fetch data

#### ETrade

- Login to ETrade.
- Hover over "At Work" in the top navbar and select "Holdings".
- Click on "View by Status".
- Click on "Download" and select "Download Expanded".
- This will download the ETrade Holdings file in XLSX format.

### Generate data for ITR

```
poetry run generate-itr-data --etrade-holdings "<path_to_file>"
```
