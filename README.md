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

##### Holdings

- Login to ETrade.
- Hoverover "At Work" in the top navbar and select "Holdings".
- Click on "View by Status".
- Click on "Download" and select "Download Expanded".
- This will download the ETrade Holdings file in XLSX format.
- The holding should be downloaded on 31st March.

##### Gains and Losses

- Login to ETrade.
- Hover over "At Work" in the top navbar and select "My Account".
- Select "Gains and Losses".
- Click on "Tax Year" and select "Custom Date".
- For "Date Range", select the 1st January to 31st March next year e.g. 1 January 2023 to 31 March 2024.
    - Note: The date values are in `MM/DD/YYYY` format.
    - Click on the calendar icon of each date to ensure you have selected the correct value.
- Ensure all other dropdowns are selected as "All".
- Ensure `Wash sale adjustment` is selected.
- Click on "Apply".
- Click on "Download" and select "Download Expanded".

##### Account Statement

- Login to ETrade.
- Hover over "Accounts" in the top navbar and hover over "Documents" and select "Brokerage Statements".
- For "Date Range", select the end date of the Indian Financial year in both date fields i.e. 31st March.
    - Note: The date values are in `MM/DD/YYYY` format.
    - Click on the calendar icon of each date to ensure you have selected the correct value.
- Click on "Apply".
- Click on "Single Account Statement" to download the statement.
- Open the file and under "Account Summary" section, find "Asset Allocation" section and note the value of "Cash". This will be used for calculatin of [Schedule AL](#schedule-al).
- Repeat this operation for fetching the account statement on 31st December which will be used in the calculation of [Schedule FA A2](#schedule-fa-a2).


### Generate data for ITR

Before running the script, ensure you check the following -

- For [holdings](#holdings) file
    - If you have downloaded the [holdings](#holdings) file during a trading window,
        - Rename the `Sellable` sheet to `Blocked`.
        - Rename the `Sellable` column to `Blocked` in `Blocked` sheet.
        - Rename the `Sellable Qty.` column to `Blocked Qty.` in `Blocked` sheet.
    - Verify that while reading the file, we need to skip the 1 row from the beginning and 4 rows at the end because they contain metadata like names, subtotals and totals. If this changes, update the `__get_shares_issued` function in `etrade.py`.
- For the [gains and losses](#gains-and-losses) file
    - Ensure the sheet is called `G&L_Expanded`.
    - Verify that while reading the file, we skip the 2 rows from the beginning and 0 rows at the end because they contain metadata like totals and subtotals. If this changes, update the `__get_shares_sold` function in `etrade.py`.
- Download the SBI Reference Rates for USD in `./data/SBI_REFERENCE_RATES_USD.csv` folder. This is to avoid downloading the files if you are running this script frequently to debug an issue. If you don't donwload this, skip the `--sbi-reference-rates` argument below and it will download it automatically for you.

Generate the data for Schedule FA A3, Schedule CG and Schedule AL(partially), by running the following command -

```
poetry run generate-itr-data --financial-year "YYYY-YYYY" --sbi-reference-rates "./data/SBI_REFERENCE_RATES_USD.csv" --etrade-holdings "<path_to_holdings_file>" --etrade-sale-transactions "<path_to_gains_and_losses_file>"
```

Follow the steps below to populate [Schedule FA A2](#schedule-fa-a2) and [Scehdule AL](#schedule-al).

Ensure you go through the logs to verify everything worked correctly.

#### Schedule FA A2

- Schedule FA A2 data is not generated from the above script. It need to be done manually.
- Using the [account statement on 31st December](#account-statement), the following would be the entries for Schedule FA A2
    - "Peak balance during the period" would be the cash in INR on 31 December. The assumption here is that you have not withdrawn the money. Else, need more thought here.
    - "Closing balance" would be the cash in INR on 31 December.
    - "Gross amount paid/credited to the account during the period" would be 0.

#### Schedule AL

- Schedule AL data is not fully generated from the above script.
- Using the [account statement on 31st March](#account-statement), add details about the cash in INR on 31st March.
- Add details about other assets and liabilities.

### FAQs

- How is reporting done for the foreign assets?
    - Stocks issued at any point that are held on 31st December are reported in Schedule FA A3.
        - The script makes an assumption that any stock held upto 31st December is not sold uptil 31st March. This will be addressed soon.
    - Stocks issued on/after 1st January but sold on/before 31st December are reported in Schedule FA A3.
    - Cash available in broker account on 31st December is reported in Schedule FA A2.
        - [Needs to be done manually](#schedule-fa-a2)
    - Stocks sold from 1st April to 31st March are reported in Schedule CG.
    - Stocks issued at any point that are held on 31st March are reported in Schedule AL.
    - Cash available in broker account on 31st March is reported in Schedule AL.
        - [Needs to be done manually](#schedule-al)
- What exchange rate is being used?
    - SBI Telegraphic Transfer Rate Buying Rate(TTBR) for USD are used to convert to INR. These values are published (almost) everyday by SBI [here](https://sbi.co.in/documents/16012/1400784/FOREX_CARD_RATES.pdf). However, it is not possible to view historical rates. https://github.com/sahilgupta/sbi-fx-ratekeeper keeps a historical record which we use.
    - For converting a stock's fair-market-value(FMV) on a particular day, we use the SBI TTBR for last day of the previous month as per the ITR rules. e.g. to convert the sale price of stock sold on 15 June 2023, the SBI TTBR for 31 May 2023. If the SBI TTBR for 31 May 2023 is not available for some reason, we subtract one day until we find a value.
    - For converting the closing value of a stock issued on year close, we use the SBI TTBR on 31st December. If the SBI TTBR for 31 December is not available for some reason, we subtract one day until we find a value.
- How is the Peak Closing Value of the stock calculated for Schedule FA A3?
    - For stocks issued but not sold as on 31st December, the maximum of the closing value of the stock from the date of issue to the 31st December - both inclusive - is the peak closing value of the stock which is then converted to INR as explained above.
    - For stocks sold before 31st December, the maximum of the closing value of the stock from the date of issue to the date of sale - both inclusive - is the peak closing value of the stock which is then converted to INR as explained above.
- How is the Closing Value of the stock calculated for Schedule FA A3?
    - For stocks issued but not sold as on 31st December, the closing value of the stock on 31st December converted to INR.
    - For stocks sold on/before 31st December, the closing value of the stock is 0.
- How is the Total gross amount paid/credited of the stock calculated for Schedule FA A3?
    - The value is 0.
- How is the Total proceeds from sale or redemption of the stock calculated for Schedule FA A3?
    - For stocks issued but not sold as on 31st December, the value is 0.
    - For stocks sold before 31st December, the value is the (sale value - issue value) which is then converted to INR as explained above.

## Disclaimer

This is strictly for my own convinience and not an advice on how to declare foreign assets. Please consult your Chartered Accountant for any advice on how to declare foreign assets.
