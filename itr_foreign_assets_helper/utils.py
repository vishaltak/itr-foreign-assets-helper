import datetime
import openpyxl
import typing


def get_ws_column_metadata(ws: openpyxl.worksheet.worksheet.Worksheet, columns: typing.List[str]) -> typing.Dict[str, str]:
    letters = {}
    for i in range(0, ws.max_column):
        cell = ws.cell(row=1, column=i+1)
        col_name = cell.value
        if col_name in columns:
            letters[cell.column_letter] = col_name
    return letters

def get_last_day_of_previous_moth(date: datetime.date) -> datetime.date:
    return date.replace(day=1) - datetime.timedelta(days=1)
