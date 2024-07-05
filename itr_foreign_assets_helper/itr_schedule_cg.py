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


class ScheduleCGRecord(itr.ScheduleRecord):
    def __init__(
            self,
            share_record: stock.ShareSoldRecord,
            sbi_reference_rates: forex.SBIReferenceRates,
            financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.share_record = share_record
        self.sbi_reference_rates = sbi_reference_rates
        self.financial_year = financial_year

        # itr fields
        self.cost_of_acquisition_without_indexation = share_record.shares_sold * \
            self.share_record.fmv_per_share_on_issue_date * \
            self.share_record.issue_date.sbi_reference_rate.tt_buy_exchange_rate
        self.cost_of_improvment_without_indexation = 0.0
        self.expenditure_wholly_and_exclusively_in_connection_with_transfer = 0.0
        self.full_value_of_consideration_received_or_receivable = share_record.shares_sold * \
            self.share_record.fmv_per_share_on_sale_date * \
            self.share_record.sale_date.sbi_reference_rate.tt_buy_exchange_rate
        self.short_term_capital_gain = self.full_value_of_consideration_received_or_receivable - \
            self.expenditure_wholly_and_exclusively_in_connection_with_transfer - \
            self.cost_of_improvment_without_indexation - \
            self.cost_of_acquisition_without_indexation
    
    def export(self) -> dict:
        return {
            'Source Metadata': json.dumps(self.share_record.source_metadata),
            'Comments': self.share_record.comments,
            'Broker': self.share_record.broker,
            'Transaction Type': self.share_record.transaction_type,
            'Award Number': self.share_record.award_number,
            'Shares Sold': self.share_record.shares_sold,
            'Issue Date': self.share_record.issue_date.actual_date,
            'FMV Per Share on Issue Date': self.share_record.fmv_per_share_on_issue_date,
            'TT Buy Rate Date Considered for Issue Date': self.share_record.issue_date.adjusted_date_for_sbi_reference_rate,
            'TT Buy Rate Considered for Issue Date': self.share_record.issue_date.sbi_reference_rate.tt_buy_exchange_rate,
            'Sale Date': self.share_record.sale_date.actual_date,
            'FMV Per Share on Sale Date': self.share_record.fmv_per_share_on_sale_date,
            'TT Buy Rate Date Considered for Sale Date': self.share_record.sale_date.adjusted_date_for_sbi_reference_rate,
            'TT Buy Rate Considered for Sale Date': self.share_record.sale_date.sbi_reference_rate.tt_buy_exchange_rate,
            'Cost of Acquisition without indexation': self.cost_of_acquisition_without_indexation,
            'Cost of Improvment without indexation': self.cost_of_improvment_without_indexation,
            'Expenditure wholly and exclusively in connection with transfer': self.expenditure_wholly_and_exclusively_in_connection_with_transfer,
            'Full value of consideration received/receivable': self.full_value_of_consideration_received_or_receivable,
            'Short term capital gain': self.short_term_capital_gain,
        }


class ScheduleCG:
    def __init__(
        self,
        shares_sold: typing.List[stock.ShareSoldRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.entries = self.__get_entries(
            shares_sold=shares_sold,
            sbi_reference_rates=sbi_reference_rates,
            financial_year=financial_year
        )
    
    def __get_entries(
        self,
        shares_sold: typing.List[stock.ShareSoldRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> typing.List[ScheduleCGRecord]:
        entries = []
        # for schedule CG, the timeframe to consider is 1 April to 31 March.
        year_start_date = financial_year[0]
        year_closing_date = financial_year[1]
        for share_sold in shares_sold:
            # ignore shares sold before year_start_date since it is outside the timeframe we are interested in for Schedule CG.
            if share_sold.sale_date.actual_date < year_start_date:
                logger.info(f'Skipping share issued on {share_sold.issue_date.actual_date} which was sold on {share_sold.sale_date.actual_date} for award number {share_sold.award_number} on {share_sold.broker} since the sale date is outside of {year_start_date} to {year_closing_date}')
                continue
            # ignore shares issued after year_closing_date since it is outside the timeframe we are interested in for Schedule CG.
            if share_sold.issue_date.actual_date > year_closing_date:
                logger.info(f'Skipping share issued on {share_sold.issue_date.actual_date} which was sold on {share_sold.sale_date.actual_date} for award number {share_sold.award_number} on {share_sold.broker} since the issue date is outside of {year_start_date} to {year_closing_date}')
                continue
            entries.append(ScheduleCGRecord(
                share_record=share_sold,
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
