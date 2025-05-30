import datetime
import logging
import json
import typing

import openpyxl
import yfinance

from . import forex
from . import itr
from . import stock
from . import utils


logger = logging.getLogger(__name__)


class ScheduleFAA3Record(itr.ScheduleRecord):
    def __init__(
            self,
            share_record: stock.ShareRecord,
            sbi_reference_rates: forex.SBIReferenceRates,
            financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.share_record = share_record
        self.sbi_reference_rates = sbi_reference_rates
        self.financial_year = financial_year
        
        # peak closing
        end_date_to_consider_for_peak_value = self.__get_end_date_to_consider_for_peak_value()
        self.peak_closing_high_date, self.peak_closing_high_value = self.__get_share_peak_closing_data(
            start_date=self.share_record.issue_date.actual_date,
            end_date=end_date_to_consider_for_peak_value
        )

        # year end
        self.year_closing_date = stock.Date(date=datetime.date(year=financial_year[0].year, month=12, day=31), type='year_closing_date', sbi_reference_rates=self.sbi_reference_rates)
        self.last_trading_date, self.fmv_per_share_on_last_trading_date = self.__get_share_year_closing_data(year_closing_date=self.year_closing_date.actual_date)

        # itr fields
        self.date_of_acquiring_interest = self.__get_date_of_acquiring_interest()
        self.initial_value_of_investment = self.__get_initial_value_of_investment()
        self.peak_value_of_investment = self.__get_peak_value_of_investment()
        self.closing_value = self.__get_closing_value()
        self.total_gross_amount_paid_or_credited_with_respect_to_holding_during_period = self.__get_total_gross_amount_paid_or_credited_with_respect_to_holding_during_period()
        self.total_proceeds_from_sale_or_redemption_of_investment_during_period = self.__get_total_proceeds_from_sale_or_redemption_of_investment_during_period()

    def __get_end_date_to_consider_for_peak_value(self) -> datetime.date:
        if self.share_record.transaction_type == 'issued':
            # since the share hasn't been sold, use the year end date - 31 december
            # since Schedule FA manages data from 1 January to 31 December
            return datetime.date(self.financial_year[0].year, month=12, day=31)
        elif self.share_record.transaction_type == 'sold':
            # since the share has been sold, use the sale date
            return self.share_record.sale_date.actual_date
        else:
            raise ValueError(f'Invalid transaction type for the share: {self.share_record.transaction_type}')

    def __get_share_peak_closing_data(self, start_date: datetime.date, end_date: datetime.date) -> typing.Tuple[stock.Date, float]:
        # end in yfinance is not inclusive. hence 1 day is added to it to get the data for that day as well.
        share_data = yfinance.download([self.share_record.ticker], start=start_date, end=end_date + datetime.timedelta(days=1))
        # extract the closing prices
        close_prices = share_data['Close']
        # TODO: for some odd reason, adding the log statement prevents crashing
        logging.debug('----')
        highest_close_date_timestamp = close_prices.idxmax().iloc[0]
        highest_close_price = close_prices.max().iloc[0]
        highest_close_date = stock.Date(date=highest_close_date_timestamp.date(), type='peak_closing_high_date', sbi_reference_rates=self.sbi_reference_rates)
        return highest_close_date, highest_close_price

    def __get_share_year_closing_data(self, year_closing_date: datetime.date) -> typing.Tuple[datetime.date, float]:
        if year_closing_date.day != 31 or year_closing_date.month != 12:
            raise ValueError(f'Invalid date for calculating the market close value at year end for ticker: {year_closing_date} . Example vlaues would be 2023-12-31')
        # the market can be closed on the 31 December.
        # to get the last close value, we consider 10 days prior to the 31 December
        # which ensures the market would be open for at least one day in that time range.
        # we then take the value for the last date i.e. 31 December if present else 30 December and so on.
        adjusted_start_date = year_closing_date - datetime.timedelta(days=10)
        # end in yfinance is not inclusive. hence 1 day is added to it to get the data for that day as well.
        share_data = yfinance.download([self.share_record.ticker], start=adjusted_start_date, end=year_closing_date + datetime.timedelta(days=1))
        # ensure the data is sorted by date
        share_data.sort_index(inplace=True)
        # TODO: for some odd reason, adding the log statement prevents crashing
        logging.debug('----')
        # check for the last available tading day on or before the year_closing_date
        last_trading_day_timestamp = max(idx for idx in share_data.index if idx.date() <= year_closing_date)
        # TODO: check if year_closing_date is null using pd.isnull(last_trading_day)
        # although highly unlikely that the market remained closed for the last 10 days of the year.
        last_trading_day_close_price = share_data.loc[last_trading_day_timestamp, 'Close'].iloc[0]
        # intentionally not returning stock.Date here since this date should not be used for TT Buy rate
        last_trading_date = last_trading_day_timestamp.date()
        return last_trading_date, last_trading_day_close_price

    def __get_date_of_acquiring_interest(self) -> datetime.date:
        return self.share_record.issue_date.actual_date
    
    def __get_initial_value_of_investment(self) -> float:
        num_of_shares = None
        if self.share_record.transaction_type == 'issued':
            num_of_shares = self.share_record.shares_issued
        elif self.share_record.transaction_type == 'sold':
            num_of_shares = self.share_record.shares_sold
        else:
            raise ValueError(f'Invalid transaction type for the share: {self.share_record.transaction_type}')
        
        return num_of_shares * \
            self.share_record.fmv_per_share_on_issue_date * \
            self.share_record.issue_date.sbi_reference_rate.tt_buy_exchange_rate
    
    def __get_peak_value_of_investment(self) -> float:
        num_of_shares = None
        if self.share_record.transaction_type == 'issued':
            num_of_shares = self.share_record.shares_issued
        elif self.share_record.transaction_type == 'sold':
            num_of_shares = self.share_record.shares_sold
        else:
            raise ValueError(f'Invalid transaction type for the share: {self.share_record.transaction_type}')
        
        return num_of_shares * \
                self.peak_closing_high_value * \
                self.peak_closing_high_date.sbi_reference_rate.tt_buy_exchange_rate
    
    def __get_closing_value(self) -> float:
        if self.share_record.transaction_type == 'issued':
            # consider the year closing date for TT Buy rate and not the trading date
            # because we want the exchange rate on 31 december of the share.
            return self.share_record.shares_issued * \
                self.fmv_per_share_on_last_trading_date * \
                self.year_closing_date.sbi_reference_rate.tt_buy_exchange_rate
        elif self.share_record.transaction_type == 'sold':
            # since the share is sold before the year closing date, it will be 0
            return 0.0
        else:
            raise ValueError(f'Invalid transaction type for the share: {self.share_record.transaction_type}')
    
    def __get_total_gross_amount_paid_or_credited_with_respect_to_holding_during_period(self) -> float:
        # since no dividend is pair, it will always be 0
        return 0.0
    
    def __get_total_proceeds_from_sale_or_redemption_of_investment_during_period(self):
        if self.share_record.transaction_type == 'issued':
            # since the share is not sold, there will be no proceeds and hence it will be 0
            return 0.0
        elif self.share_record.transaction_type == 'sold':
            return self.share_record.shares_sold * \
                self.share_record.fmv_per_share_on_sale_date * \
                self.share_record.sale_date.sbi_reference_rate.tt_buy_exchange_rate
        else:
            raise ValueError(f'Invalid transaction type for the share: {self.share_record.transaction_type}')
    
    def export(self) -> dict:
        return {
            'Source Metadata': json.dumps(self.share_record.source_metadata),
            'Comments': self.share_record.comments,
            'Broker': self.share_record.broker,
            'Transaction Type': self.share_record.transaction_type,
            'Award Number': self.share_record.award_number,
            'Shares Issued': utils.rercursive_getattr(self.share_record, 'shares_issued', None),
            'Shares Sold': utils.rercursive_getattr(self.share_record, 'shares_sold', None),
            'Issue Date': self.share_record.issue_date.actual_date,
            'FMV Per Share on Issue Date': self.share_record.fmv_per_share_on_issue_date,
            'TT Buy Rate Date Considered for Issue Date': self.share_record.issue_date.adjusted_date_for_sbi_reference_rate,
            'TT Buy Rate Considered for Issue Date': self.share_record.issue_date.sbi_reference_rate.tt_buy_exchange_rate,
            'Sale Date': utils.rercursive_getattr(self.share_record, 'sale_date.actual_date', None),
            'FMV Per Share on Sale Date': utils.rercursive_getattr(self.share_record, 'fmv_per_share_on_sale_date', None),
            'TT Buy Rate Date Considered for Sale Date': utils.rercursive_getattr(self.share_record, 'sale_date.adjusted_date_for_sbi_reference_rate', None),
            'TT Buy Rate Considered for Sale Date': utils.rercursive_getattr(self.share_record, 'sale_date.sbi_reference_rate.tt_buy_exchange_rate', None),
            'Peak Closing High Date': self.peak_closing_high_date.actual_date,
            'Peak Closing High Value': self.peak_closing_high_value,
            'TT Buy Rate Date Considered for Peak Closing High Date': self.peak_closing_high_date.adjusted_date_for_sbi_reference_rate,
            'TT Buy Rate Considered for Peak Closing High Date': self.peak_closing_high_date.sbi_reference_rate.tt_buy_exchange_rate,
            'Year Closing Date': self.year_closing_date.actual_date,
            'Last Trading Date': self.last_trading_date,
            'FMV Per Share on Last Trading Date': self.fmv_per_share_on_last_trading_date,
            'TT Buy Rate Date Considered for Year Closing Date': self.year_closing_date.adjusted_date_for_sbi_reference_rate,
            'TT Buy Rate Considered for Year Closing Date': self.year_closing_date.sbi_reference_rate.tt_buy_exchange_rate,
            'Date of Acquiring Interest': self.date_of_acquiring_interest,
            'Initial Value of Investment': self.initial_value_of_investment,
            'Peak Value of Investment': self.peak_value_of_investment,
            'Closing Value': self.closing_value,
            'Total gross amount paid/credited with respect to the holding during the period': self.total_gross_amount_paid_or_credited_with_respect_to_holding_during_period,
            'Total gross proceeds from sale or redemption of investment during the period': self.total_proceeds_from_sale_or_redemption_of_investment_during_period,
        }


class ScheduleFAA3:
    def __init__(
        self,
        shares_issued: typing.List[stock.ShareIssuedRecord],
        shares_sold: typing.List[stock.ShareSoldRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.entries = self.__get_entries(
            shares_issued=shares_issued,
            shares_sold=shares_sold,
            sbi_reference_rates=sbi_reference_rates,
            financial_year=financial_year
        )
    
    def __get_entries(
        self,
        shares_issued: typing.List[stock.ShareIssuedRecord],
        shares_sold: typing.List[stock.ShareSoldRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> typing.List[ScheduleFAA3Record]:
        entries = []
        # for schedule FA, the timeframe to consider is 1 January to 31 December.
        year_start_date = datetime.date(year=financial_year[0].year, month=1, day=1)
        year_closing_date = datetime.date(year=financial_year[0].year, month=12, day=31)
        for share_issued in shares_issued:
            # ignore shares issued after year_closing_date since it is outside the timeframe we are interested in for Schedule FA.
            # do not compare the issue date with the year_start_date since we want to report all holdings we have upto this point regardless of when it was issued.
            if share_issued.issue_date.actual_date > year_closing_date:
                logger.info(f'Skipping share issued on {share_issued.issue_date.actual_date} for award number {share_issued.award_number} on {share_issued.broker} since the issue date is outside of {year_start_date} to {year_closing_date}')
                continue
            entries.append(ScheduleFAA3Record(
                share_record=share_issued,
                sbi_reference_rates=sbi_reference_rates,
                financial_year=financial_year,
            ))
        for share_sold in shares_sold:
            # ignore shares sold before year_start_date since it is outside the timeframe we are interested in for Schedule FA.
            if share_sold.sale_date.actual_date < year_start_date:
                logger.info(f'Skipping share issued on {share_sold.issue_date.actual_date} which was sold on {share_sold.sale_date.actual_date} for award number {share_sold.award_number} on {share_sold.broker} since the sale date is outside of {year_start_date} to {year_closing_date}')
                continue
            # ignore shares issued after year_closing_date since it is outside the timeframe we are interested in for Schedule FA.
            if share_sold.issue_date.actual_date > year_closing_date:
                logger.info(f'Skipping share issued on {share_sold.issue_date.actual_date} which was sold on {share_sold.sale_date.actual_date} for award number {share_sold.award_number} on {share_sold.broker} since the issue date is outside of {year_start_date} to {year_closing_date}')
                continue
            entries.append(ScheduleFAA3Record(
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
