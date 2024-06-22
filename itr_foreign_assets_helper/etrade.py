import datetime
import logging
import typing

import openpyxl
import openpyxl.worksheet
import openpyxl.worksheet.worksheet

from pathlib import Path

from . import stock
from . import utils


logger = logging.getLogger(__name__)


class ETradeTransactions:
    
    def __init__(self, holdings_file: typing.IO, sale_transactions_file: typing.IO) -> None:
        self.broker = "ETrade"
        self.stocks_released = self.__get_stocks_released(holdings_file=holdings_file)
        self.stocks_sold = self.__get_stocks_sold(sale_transactions_file=sale_transactions_file)
        self.cash = self.__get_cash(holdings_file=holdings_file)
    
    def __extract_data(
            self,
            file: typing.IO,
            sheet_name: str,
            row_to_skip_at_start: int,
            row_to_skip_at_end: int,
            required_columns: typing.Dict[str, str]
    ) -> typing.Dict:
        file_name = Path(file.name).name
        wb = openpyxl.load_workbook(file)
        ws = wb[sheet_name]
        required_column_letters = utils.get_ws_column_metadata(
            ws=ws,
            columns=required_columns.keys(),
        )
        data = []
        # mix_row and max_row are 1-based index.
        min_row = ws.min_row + row_to_skip_at_start
        max_row = ws.max_row - row_to_skip_at_end
        curr_row_num = min_row
        for row in ws.iter_rows(min_row=min_row, max_row=max_row):
            curr_row_data = {}
            curr_row_data['source_metadata'] = {
                'file_name': file_name,
                'sheet_name': sheet_name,
                'row': curr_row_num,
            }
            for cell in row:
                if cell.column_letter in required_column_letters:
                    cell_title = required_column_letters[cell.column_letter]
                    translated_title = required_columns[cell_title]
                    curr_row_data[translated_title] = cell.value
            data.append(curr_row_data)
            curr_row_num += 1
        return data
    
    def __get_stocks_released(self, holdings_file: typing.IO) -> typing.List[stock.StockReleasedRecord]:
        sheet_name = 'Sellable'
        required_columns = {
            'Symbol': 'ticker',
            'Sellable Qty.': 'shares_issued',
            'Grant Number': 'award_number',
            'Release Date': 'release_date',
            'Purchase Date FMV': 'market_value_per_share',
        }
        stocks_released = []
        # the first row is the column names and hence skipped
        row_to_skip_at_start = 1
        # the last row is the total and hence skipped
        row_to_skip_at_end = 1
        raw_stocks_data = self.__extract_data(
            file=holdings_file,
            sheet_name=sheet_name,
            required_columns=required_columns,
            row_to_skip_at_start=row_to_skip_at_start,
            row_to_skip_at_end=row_to_skip_at_end,
        )
        for stock_data in raw_stocks_data:
            stock_released = stock.StockReleasedRecord(
                source_metadata=stock_data['source_metadata'],
                broker=self.broker,
                comments='',
                ticker=stock_data['ticker'],
                award_number=int(stock_data['award_number']),
                shares_issued=stock_data['shares_issued'],
                release_date=datetime.datetime.strptime(stock_data['release_date'], '%d-%b-%Y').date(),
                market_value_per_share=float(stock_data['market_value_per_share'].replace('$', '')),
            )
            logger.debug(stock_released)
            stocks_released.append(stock_released)
        return stocks_released

    def __get_stocks_sold(self, sale_transactions_file: typing.IO) -> typing.List[stock.StockSoldRecord]:
        sheet_name = 'G&L_Expanded'
        required_columns = {
            'Symbol': 'ticker',
            'Qty.': 'shares_sold',
            'Grant Number': 'award_number',
            'Date Acquired': 'release_date',
            'Vest Date FMV': 'market_value_per_share',
            'Date Sold': 'sale_date',
            'Proceeds Per Share': 'sale_value_per_share',
            'Order Number': 'sale_order_number'
        }
        stocks_sold = []
        # the first row is the column names and hence skipped
        # the second row is a summary rpw and hence skipped
        row_to_skip_at_start = 2
        # there is no extra row at the end. hence no rows skipped.
        row_to_skip_at_end = 0
        raw_stocks_data = self.__extract_data(
            file=sale_transactions_file,
            sheet_name=sheet_name,
            required_columns=required_columns,
            row_to_skip_at_start=row_to_skip_at_start,
            row_to_skip_at_end=row_to_skip_at_end,
        )
        for stock_data in raw_stocks_data:
            stock_sold = stock.StockSoldRecord(
                source_metadata=stock_data['source_metadata'],
                broker=self.broker,
                comments='',
                ticker=stock_data['ticker'],
                award_number=int(stock_data['award_number']),
                release_date=datetime.datetime.strptime(stock_data['release_date'], '%m/%d/%Y').date(),
                market_value_per_share=float(str(stock_data['market_value_per_share']).replace('$', '')),
                shares_sold=stock_data['shares_sold'],
                sale_date=datetime.datetime.strptime(stock_data['sale_date'], '%m/%d/%Y').date(),
                sale_value_per_share=float(str(stock_data['sale_value_per_share']).replace('$', '')),
            )
            logger.debug(stock_sold)
            stocks_sold.append(stock_sold)
        return stocks_sold
    
    def __get_cash(self, holdings_file: typing.IO) -> stock.CashRecord:
        sheet_name = 'Other Holdings'
        required_columns = {
            'Est. Market Value': 'amount',
        }
        cash = None
        # the first row is the column names and hence skipped
        # the second row is empty with lines and hence skipped
        row_to_skip_at_start = 2
        # the last row caontains the cash as total
        row_to_skip_at_end = 0
        raw_cash_data = self.__extract_data(
            file=holdings_file,
            sheet_name=sheet_name,
            required_columns=required_columns,
            row_to_skip_at_start=row_to_skip_at_start,
            row_to_skip_at_end=row_to_skip_at_end,
        )
        if len(raw_cash_data) != 1:
            raise Exception(f'Failed to extract cash holdings for ${self.broker}')
        cash = stock.CashRecord(
            source_metadata=raw_cash_data[0]['source_metadata'],
            broker=self.broker,
            comments='',
            amount=float(str(raw_cash_data[0]['amount']).replace('$', '')),
        )
        logger.debug(cash)
        return cash
