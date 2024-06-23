import datetime
import logging
import typing

import yfinance

from . import forex
from . import itr
from . import stock


logger = logging.getLogger(__name__)


class ScheduleFAA2Record(itr.ScheduleRecord):
    def __init__(
            self,
            cash: stock.CashRecord,
            sbi_reference_rates: forex.SBIReferenceRates,
            financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.source_metadata = cash.source_metadata
        self.broker = cash.broker
        self.comments = cash.comments
        # for financial year 2023-2024, the year_closing_date would be 2023-12-31 since Schedule FA records are from January 1 to December 31.
        self.year_closing_date = datetime.date(year=financial_year[0].year, month=12, day=31)
        # for year_closing_date_adjusted_for_tt_buy_rate , if the sbi_reference_rates.on_date for that day is missing, it will keep subtracting one day till it finds a exchange rate.
        self.year_closing_date_adjusted_for_tt_buy_rate, self.tt_buy_rate_for_year_closing_date = sbi_reference_rates.on_date(self.year_closing_date, adjust_to_last_day_of_previous_month=False)

        # TODO: need to think about this more
        # the final fields that are required while filing the ITR
        self.peak_balance_during_period = cash.amount * self.tt_buy_rate_for_year_closing_date.tt_buy_exchange_rate
        self.closing_balance = cash.amount * self.tt_buy_rate_for_year_closing_date.tt_buy_exchange_rate
        self.total_gross_amount_paid_or_credited_to_account_during_period = 0.0
        self.nature_of_income = 'Not Applicable'


class ScheduleFAA2:
    def __init__(
        self,
        cash: stock.CashRecord,
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.entries = self.__get_entries(
            cash=cash,
            sbi_reference_rates=sbi_reference_rates,
            financial_year=financial_year
        )
    
    def __get_entries(
        self,
        cash: stock.CashRecord,
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> typing.List[ScheduleFAA2Record]:
        entries = []
        # for schedule FA, the timeframe to consider is 1 January to 31 December.
        entries.append(ScheduleFAA2Record(
            cash=cash,
            sbi_reference_rates=sbi_reference_rates,
            financial_year=financial_year,
        ))
        return entries


class ScheduleFAA3Record(itr.ScheduleRecord):
    def __init__(
            self,
            stock: stock.ShareRecord,
            sbi_reference_rates: forex.SBIReferenceRates,
            financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        # TODO: move this entire logic to the stock classes
        super().__init__()
        self.source_metadata = stock.source_metadata
        self.broker = stock.broker
        self.comments = stock.comments
        self.ticker = stock.ticker
        self.award_number = stock.award_number
        self.transaction_type = stock.transaction_type
        self.peak_closing_high_date = None
        self.peak_closing_high_value = None
        # following attributes are set in __set_release_date_metadata()
        self.shares_released = None
        self.release_date = None
        self.market_value_per_share = None
        self.release_date_adjusted_for_tt_buy_rate = None
        self.tt_buy_rate_for_release_date = None
        # following attributes are set in __set_sale_date_metadata()
        self.shares_sold = None
        self.sale_date = None
        self.sale_value_per_share = None
        self.release_date_adjusted_for_tt_buy_rate = None
        self.tt_buy_rate_for_release_date = None
        # the following attributes are set in __set_year_end_metadata()
        self.year_closing_date = None
        self.year_closing_date_adjusted_for_market_closure = None
        self.market_value_per_share_on_year_closing_date = None
        self.year_closing_date_adjusted_for_tt_buy_rate = None
        self.tt_buy_rate_for_year_closing_date = None
        # the following attributes are set in __set_peak_date_metadata()
        self.peak_closing_high_date = None
        self.peak_closing_high_value = None
        self.peak_closing_high_date_adjusted_for_tt_buy_rate = None
        self.tt_buy_rate_for_peak_closing_high_date = None

        # depending on the tranction_type of the stock, set the approrpriate attributes.
        if self.transaction_type == 'released':
            self.shares_released = stock.shares_released
            self.__set_release_date_metadata(stock=stock, sbi_reference_rates=sbi_reference_rates)
            self.__set_year_end_metadata(sbi_reference_rates=sbi_reference_rates, financial_year=financial_year)
            # for financial year 2023-2024, the end_date to consider for peak closing for
            # stock that are held would be year_closing_date e.g. 2023-12-31 .
            # since Schedule FA records are from January 1 to December 31.
            self.__set_peak_date_metadata(start_date=self.release_date, end_date=self.year_closing_date, sbi_reference_rates=sbi_reference_rates)
            # the final fields that are required while filing the ITR
            self.date_of_acquiring_interest = self.release_date
            self.initial_value_of_investment = self.shares_released * self.market_value_per_share * self.tt_buy_rate_for_release_date.tt_buy_exchange_rate
            self.peak_value_of_investment_during_period = self.shares_released * self.peak_closing_high_value * self.tt_buy_rate_for_peak_closing_high_date.tt_buy_exchange_rate
            self.closing_value = self.shares_released * self.market_value_per_share_on_year_closing_date * self.tt_buy_rate_for_year_closing_date.tt_buy_exchange_rate
            # since there is no dividend being paid, it is set to 0.0
            self.total_gross_amount_paid_or_credited_to_account_during_period = 0.0
            # since there is no sale and thus no proceed, it is set to 0.0
            self.total_gross_proceeds_from_sale_or_redemption_of_investment_during_period = 0.0
        elif self.transaction_type == 'sold':
            self.shares_sold = stock.shares_sold
            self.__set_release_date_metadata(stock=stock, sbi_reference_rates=sbi_reference_rates)
            self.__set_sale_date_metadata(stock=stock, sbi_reference_rates=sbi_reference_rates)
            # since th stock was sold, the end_date to consider for peak closing for
            # stock would be the sale_date of the stock.
            self.__set_peak_date_metadata(start_date=self.release_date, end_date=self.sale_date, sbi_reference_rates=sbi_reference_rates)
            # the final fields that are required while filing the ITR
            self.date_of_acquiring_interest = self.release_date
            self.initial_value_of_investment = self.shares_sold * self.market_value_per_share * self.tt_buy_rate_for_release_date.tt_buy_exchange_rate
            self.peak_value_of_investment_during_period = self.shares_sold * self.peak_closing_high_value * self.tt_buy_rate_for_peak_closing_high_date.tt_buy_exchange_rate
            # since there is a sale and thus not retained at end of year, it is set to 0.0
            self.closing_value = 0.0
            # since there is no dividend being paid, it is set to 0.0
            self.total_gross_amount_paid_or_credited_to_account_during_period = 0.0
            self.total_gross_proceeds_from_sale_or_redemption_of_investment_during_period = self.shares_sold * self.sale_value_per_share * self.tt_buy_rate_for_sale_date.tt_buy_exchange_rate
        else:
            raise ValueError(f'Invalid transaction_type: {self.transaction_type}')
    

    def __set_release_date_metadata(
        self,
        stock: stock.ShareRecord,
        sbi_reference_rates: forex.SBIReferenceRates,
    ) -> None:
        self.release_date = stock.release_date
        self.market_value_per_share = stock.market_value_per_share
        # release_date_adjusted_for_tt_buy_rate will be the last day of the previous month.
        # however, if the sbi_reference_rates.on_date for that day is missing, it will keep subtracting one day till it finds a exchange rate.
        self.release_date_adjusted_for_tt_buy_rate, self.tt_buy_rate_for_release_date = sbi_reference_rates.on_date(self.release_date, adjust_to_last_day_of_previous_month=True)
    
    def __set_sale_date_metadata(
        self,
        stock: stock.ShareRecord,
        sbi_reference_rates: forex.SBIReferenceRates,
    ) -> None:
        self.sale_date = stock.sale_date
        self.sale_value_per_share = stock.sale_value_per_share
        # sale_date_adjusted_for_tt_buy_rate will be the last day of the previous month.
        # however, if the sbi_reference_rates.on_date for that day is missing, it will keep subtracting one day till it finds a exchange rate.
        self.sale_date_adjusted_for_tt_buy_rate, self.tt_buy_rate_for_sale_date = sbi_reference_rates.on_date(self.sale_date, adjust_to_last_day_of_previous_month=True)

    def __set_year_end_metadata(
            self,
            sbi_reference_rates: forex.SBIReferenceRates,
            financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        # for financial year 2023-2024, the year_closing_date would be 2023-12-31 since Schedule FA records are from January 1 to December 31.
        self.year_closing_date = datetime.date(year=financial_year[0].year, month=12, day=31)
        self.year_closing_date_adjusted_for_market_closure, self.market_value_per_share_on_year_closing_date = self.__get_stock_year_closing_data(year_closing_date=self.year_closing_date)
        # for year_closing_date_adjusted_for_tt_buy_rate , if the sbi_reference_rates.on_date for that day is missing, it will keep subtracting one day till it finds a exchange rate.
        self.year_closing_date_adjusted_for_tt_buy_rate, self.tt_buy_rate_for_year_closing_date = sbi_reference_rates.on_date(self.year_closing_date_adjusted_for_market_closure, adjust_to_last_day_of_previous_month=False)

    def __set_peak_date_metadata(
        self,
        start_date: datetime.date,
        end_date: datetime.date,
        sbi_reference_rates: forex.SBIReferenceRates,
    ) -> None:
        self.peak_closing_high_date, self.peak_closing_high_value = self.__get_stock_peak_closing_data(start_date=start_date, end_date=end_date)
        # peak_closing_high_date_adjusted_for_tt_buy_rate will be the last day of the previous month.
        # however, if the sbi_reference_rates.on_date for that day is missing, it will keep subtracting one day till it finds a exchange rate.
        self.peak_closing_high_date_adjusted_for_tt_buy_rate, self.tt_buy_rate_for_peak_closing_high_date = sbi_reference_rates.on_date(self.peak_closing_high_date, adjust_to_last_day_of_previous_month=True)

    def __get_stock_year_closing_data(self, year_closing_date: datetime.date) -> typing.Tuple[datetime.date, float]:
        if year_closing_date.day != 31 or year_closing_date.month != 12:
            raise Exception(f'Invalid date for calculating the market close value at year end for ticker: {year_closing_date} . Example vlaues would be 2023-12-31')
        # the market can be closed on the 31 December.
        # to get the last close value, we consider 10 days prior to the 31 December
        # which ensures the market would be open for at least one day in that time range.
        # we then take the value for the last date i.e. 31 December if present else 30 December and so on.
        adjusted_start_date = year_closing_date - datetime.timedelta(days=10)
        # end in yfinance is not inclusive. hence 1 day is added to it to get the data for that day as well.
        stock_data = yfinance.download([self.ticker], start=adjusted_start_date, end=year_closing_date + datetime.timedelta(days=1))
        # ensure the data is sorted by date
        stock_data.sort_index(inplace=True)
        # TODO: for some odd reason, adding the log statement prevents crashing
        logging.debug('----')
        # check for the last available tading day on or before the year_closing_date
        last_trading_day_timestamp = max(idx for idx in stock_data.index if idx.date() <= year_closing_date)
        # TODO: check if year_closing_date is null using pd.isnull(last_trading_day)
        # although highly unlikely that the market remained closed for the last 10 days of the year.
        last_trading_day_close_price = stock_data.loc[last_trading_day_timestamp, 'Close']
        return last_trading_day_timestamp.date(), last_trading_day_close_price

    def __get_stock_peak_closing_data(self, start_date: datetime.date, end_date: datetime.date) -> typing.Tuple[datetime.date, float]:
        # end in yfinance is not inclusive. hence 1 day is added to it to get the data for that day as well.
        stock_data = yfinance.download([self.ticker], start=start_date, end=end_date + datetime.timedelta(days=1))
        # extract the closing prices
        close_prices = stock_data['Close']
        highest_close_date = close_prices.idxmax()
        highest_close_price = close_prices.max()
        return highest_close_date.date(), highest_close_price


class ScheduleFAA3:
    def __init__(
        self,
        shares_released: typing.List[stock.ShareReleasedRecord],
        shares_sold: typing.List[stock.ShareSoldRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.entries = self.__get_entries(
            shares_released=shares_released,
            shares_sold=shares_sold,
            sbi_reference_rates=sbi_reference_rates,
            financial_year=financial_year
        )
    
    def __get_entries(
        self,
        shares_released: typing.List[stock.ShareReleasedRecord],
        shares_sold: typing.List[stock.ShareSoldRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> typing.List[ScheduleFAA3Record]:
        entries = []
        # for schedule FA, the timeframe to consider is 1 January to 31 December.
        year_start_date = datetime.date(year=financial_year[0].year, month=1, day=1)
        year_closing_date = datetime.date(year=financial_year[0].year, month=12, day=31)
        for share_released in shares_released:
            # ignore stocks release after year_closing_date since it is outisde the timeframe we are interested in.
            if share_released.release_date > year_closing_date:
                logger.info(f'Skipping stock released on {share_released.release_date} for award number {share_released.award_number} on {share_released.broker} since it has occured after {year_closing_date}')
                continue
            entries.append(ScheduleFAA3Record(
                stock=share_released,
                sbi_reference_rates=sbi_reference_rates,
                financial_year=financial_year,
            ))
        for share_sold in shares_sold:
            # ignore stocks sold before year_start_date since it is outisde the timeframe we are interested in.
            # should not compare the release date for these stocks with year_end_date because the stocks can be bought at any time in the past.
            if share_sold.sale_date < year_start_date:
                logger.info(f'Skipping stock sold on {share_sold.sale_date} for award number {share_sold.award_number} on {share_sold.broker} since it has occured before {year_start_date}')
                continue
            entries.append(ScheduleFAA3Record(
                stock=share_sold,
                sbi_reference_rates=sbi_reference_rates,
                financial_year=financial_year,
            ))
        return entries
