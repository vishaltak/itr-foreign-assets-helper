import datetime
import functools
import openpyxl
import typing


def get_ws_column_metadata(ws: openpyxl.worksheet.worksheet.Worksheet, columns: typing.List[str]) -> typing.Dict[str, str]:
    letters = {}
    for i in range(0, ws.max_column):
        cell = ws.cell(row=1, column=i+1)
        col_name = cell.value
        if col_name in columns:
            if col_name in letters.values():
                raise ValueError(f"Duplicate column name '{col_name}' in sheet '{ws}'")
            letters[cell.column_letter] = col_name
    return letters

def get_last_day_of_previous_moth(date: datetime.date) -> datetime.date:
    return date.replace(day=1) - datetime.timedelta(days=1)


# Source - https://stackoverflow.com/a/31174427
def rercursive_setattr(obj, attr, val):
    pre, _, post = attr.rpartition('.')
    return setattr(rercursive_getattr(obj, pre) if pre else obj, post, val)

def rercursive_getattr(obj, attr, *args):
    def _getattr(obj, attr):
        return getattr(obj, attr, *args)
    return functools.reduce(_getattr, [obj] + attr.split('.'))