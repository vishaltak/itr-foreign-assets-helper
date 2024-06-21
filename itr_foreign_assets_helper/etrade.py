import logging
import typing

import openpyxl

logger = logging.getLogger(__name__)

def convert(holdings_file: typing.IO) -> openpyxl.Workbook:
    wb = openpyxl.load_workbook(holdings_file)
    return wb
