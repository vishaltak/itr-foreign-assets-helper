import datetime
import logging
import typing

import openpyxl
import openpyxl.worksheet
import openpyxl.worksheet.worksheet

from pathlib import Path

from . import platform
from . import stock
from . import utils

logger = logging.getLogger(__name__)

class ETrade(platform.Platform):
    def __init__(self) -> None:
        self.name = "ETrade"
    
    def get_shares_issued(self, holdings_file: typing.IO) -> typing.List[stock.StockIssued]:
        wb = openpyxl.load_workbook(holdings_file)
        file_name = Path(holdings_file.name).name
        sheet_name = 'Sellable'
        ws = wb[sheet_name]
        required_columns = {
            'Symbol': 'symbol',
            'Sellable Qty.': 'shares_issued',
            'Grant Number': 'award_number',
            'Release Date': 'release_date',
            'Purchase Date FMV': 'market_value_per_share',
        }
        required_column_letters = utils.get_ws_column_metadata(
            ws=ws,
            columns=required_columns.keys(),
        )
        # mix_row is 1-based index. the first row is the column names and hence skipped
        # max_row is 1-based index. the last row is the total and hence skipped
        curr_row_num = 2
        for row in ws.iter_rows(min_row=curr_row_num, max_row=ws.max_row - 1):
            transaction_data = {}
            for cell in row:
                if cell.column_letter in required_column_letters:
                    cell_title = required_column_letters[cell.column_letter]
                    translated_title = required_columns[cell_title]
                    transaction_data[translated_title] = cell.value
            stock_issued = stock.StockIssued(
                source_metadata= {
                    'file_name': file_name,
                    'sheet_name': sheet_name,
                    'row': curr_row_num,
                },
                platform=self.name,
                comments="",
                award_number=int(transaction_data['award_number']),
                shares_issued=transaction_data['shares_issued'],
                release_date=datetime.datetime.strptime(transaction_data['release_date'], '%d-%b-%Y').date(),
                market_value_per_share=float(transaction_data['market_value_per_share'].replace('$', '')),
            )
            logger.debug(stock_issued)
            curr_row_num += 1
        return wb
