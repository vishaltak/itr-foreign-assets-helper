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
    def __init__(self, holdings_file: typing.IO) -> None:
        self.broker = "ETrade"
        self.stocks_released = self.__get_stocks_released(holdings_file=holdings_file)
        self.stocks_sold = self.__get_stocks_sold()
    
    def __get_stocks_released(self, holdings_file: typing.IO) -> typing.List[stock.StockReleasedRecord]:
        wb = openpyxl.load_workbook(holdings_file)
        file_name = Path(holdings_file.name).name
        sheet_name = 'Sellable'
        ws = wb[sheet_name]
        required_columns = {
            'Symbol': 'ticker',
            'Sellable Qty.': 'shares_issued',
            'Grant Number': 'award_number',
            'Release Date': 'release_date',
            'Purchase Date FMV': 'market_value_per_share',
        }
        required_column_letters = utils.get_ws_column_metadata(
            ws=ws,
            columns=required_columns.keys(),
        )
        stocks_released = []
        # mix_row is 1-based index. the first row is the column names and hence skipped
        # max_row is 1-based index. the last row is the total and hence skipped
        curr_row_num = 2
        for row in ws.iter_rows(min_row=curr_row_num, max_row=ws.max_row - 1):
            raw_data = {}
            for cell in row:
                if cell.column_letter in required_column_letters:
                    cell_title = required_column_letters[cell.column_letter]
                    translated_title = required_columns[cell_title]
                    raw_data[translated_title] = cell.value
            stock_released = stock.StockReleasedRecord(
                source_metadata={
                    'file_name': file_name,
                    'sheet_name': sheet_name,
                    'row': curr_row_num,
                },
                broker=self.broker,
                comments='',
                ticker=raw_data['ticker'],
                award_number=int(raw_data['award_number']),
                shares_issued=raw_data['shares_issued'],
                release_date=datetime.datetime.strptime(raw_data['release_date'], '%d-%b-%Y').date(),
                market_value_per_share=float(raw_data['market_value_per_share'].replace('$', '')),
            )
            logger.debug(stock_released)
            stocks_released.append(stock_released)
            curr_row_num += 1
        return stocks_released

    def __get_stocks_sold(self) -> typing.List[stock.StockSoldRecord]:
        return []
