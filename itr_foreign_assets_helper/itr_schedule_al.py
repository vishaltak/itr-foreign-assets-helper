import datetime
import logging
import json
import typing

import openpyxl

from . import forex
from . import itr
from . import stock
from . import utils


logger = logging.getLogger(__name__)


class ScheduleALRecord(itr.ScheduleRecord):
    def __init__(
            self,
            share_record: stock.ShareIssuedRecord,
            sbi_reference_rates: forex.SBIReferenceRates,
            financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.share_record = share_record
        self.sbi_reference_rates = sbi_reference_rates
        self.financial_year = financial_year

        # itr fields
        self.cost_of_acquisition_without_indexation = share_record.shares_issued * \
            self.share_record.fmv_per_share_on_issue_date * \
            self.share_record.issue_date.sbi_reference_rate.tt_buy_exchange_rate
        self.cost_of_improvment_without_indexation = 0.0
    
    def export(self) -> dict:
        return {
            'Source Metadata': json.dumps(self.share_record.source_metadata),
            'Comments': self.share_record.comments,
            'Broker': self.share_record.broker,
            'Transaction Type': self.share_record.transaction_type,
            'Award Number': self.share_record.award_number,
            'Shares Issued': self.share_record.shares_issued,
            'Issue Date': self.share_record.issue_date.actual_date,
            'FMV Per Share on Issue Date': self.share_record.fmv_per_share_on_issue_date,
            'TT Buy Rate Date Considered for Issue Date': self.share_record.issue_date.adjusted_date_for_sbi_reference_rate,
            'TT Buy Rate Considered for Issue Date': self.share_record.issue_date.sbi_reference_rate.tt_buy_exchange_rate,
            'Cost of Acquisition without indexation': self.cost_of_acquisition_without_indexation,
        }


class ScheduleAL:
    def __init__(
        self,
        shares_issued: typing.List[stock.ShareIssuedRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.entries = self.__get_entries(
            shares_issued=shares_issued,
            sbi_reference_rates=sbi_reference_rates,
            financial_year=financial_year
        )
    
    def __get_entries(
        self,
        shares_issued: typing.List[stock.ShareIssuedRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> typing.List[ScheduleALRecord]:
        entries = []
        # for schedule AL, the timeframe to consider is 1 April to 31 March.
        year_start_date = financial_year[0]
        year_closing_date = financial_year[1]
        for share_issued in shares_issued:
            # ignore shares issued after year_closing_date since it is outside the timeframe we are interested in for Schedule AL.
            # do not compare the issue date with the year_start_date since we want to report all holdings we have upto this point regardless of when it was issued.
            if share_issued.issue_date.actual_date > year_closing_date:
                logger.info(f'Skipping share issued on {share_issued.issue_date.actual_date} for award number {share_issued.award_number} on {share_issued.broker} since the issue date is outside of {year_start_date} to {year_closing_date}')
                continue
            entries.append(ScheduleALRecord(
                share_record=share_issued,
                sbi_reference_rates=sbi_reference_rates,
                financial_year=financial_year,
            ))
        return entries
    
    def export(self, workbook: openpyxl.Workbook, sheet_name: str) -> openpyxl.Workbook:
        ws = workbook[sheet_name]
        title_added = False
        for entry in self.entries:
            data = entry.export()
            if not title_added:
                ws.append(list(data.keys()))
                title_added = True
            ws.append(list(data.values()))
        return workbook
