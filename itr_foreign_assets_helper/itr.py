import datetime
import logging
import typing

import yfinance

from . import forex
from . import stock


logger = logging.getLogger(__name__)


class ScheduleRecord:

    def __repr__(self) -> str:
        attrs = ', '.join(f'{key}={value!r}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'
    
    def __str__(self) -> str:
        attrs = ', '.join(f'{key}={value}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'


class ScheduleFAA3Record(ScheduleRecord):
    def __init__(
            self,
            stock: stock.StockRecord,
            sbi_reference_rates: forex.SBIReferenceRates,
            financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
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
        self.shares_issued = None
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

        # depending on the tranction_type of the stock, set the approrpriate attributes.
        if self.transaction_type == 'released':
            self.shares_issued = stock.shares_issued
            self.__set_release_date_metadata(stock=stock, sbi_reference_rates=sbi_reference_rates)
            self.__set_year_end_metadata(sbi_reference_rates=sbi_reference_rates, financial_year=financial_year)
            # for financial year 2023-2024, the end_date to consider for peak closing for
            # stock that are held would be year_closing_date e.g. 2023-12-31 .
            # since Schedule FA records are from January 1 to December 31.
            self.peak_closing_high_date, self.peak_closing_high_value = self.__get_stock_peak_closing_data(start_date=self.release_date, end_date=self.year_closing_date)
        elif self.transaction_type == 'sold':
            self.shares_sold = stock.shares_sold
            self.__set_release_date_metadata(stock=stock, sbi_reference_rates=sbi_reference_rates)
            self.__set_sale_date_metadata(stock=stock, sbi_reference_rates=sbi_reference_rates)
            # since th stock was sold, the end_date to consider for peak closing for
            # stock would be the sale_date of the stock.
            self.peak_closing_high_date, self.peak_closing_high_value = self.__get_stock_peak_closing_data(start_date=self.release_date, end_date=self.sale_date)
        else:
            raise ValueError(f'Invalid transaction_type: ${self.transaction_type}')
    
    def __set_release_date_metadata(
        self,
        stock: stock.StockRecord,
        sbi_reference_rates: forex.SBIReferenceRates,
    ) -> None:
        self.release_date = stock.release_date
        self.market_value_per_share = stock.market_value_per_share
        # release_date_adjusted_for_tt_buy_rate will be the last day of the previous month.
        # however, if the sbi_reference_rates.on_date for that day is missing, it will keep subtracting one day till it finds a exchange rate.
        self.release_date_adjusted_for_tt_buy_rate, self.tt_buy_rate_for_release_date = sbi_reference_rates.on_date(self.release_date, adjust_to_last_day_of_previous_month=True)
    
    def __set_sale_date_metadata(
        self,
        stock: stock.StockRecord,
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

    def __get_stock_year_closing_data(self, year_closing_date: datetime.date) -> typing.Tuple[datetime.date, float]:
        if year_closing_date.day != 31 or year_closing_date.month != 12:
            raise Exception(f'Invalid date for calculating the market close value at year end for ticker: ${year_closing_date} . Example vlaues would be 2023-12-31')
        # the market can be closed on the 31 December.
        # to get the last close value, we consider 10 days prior to the 31 December
        # which ensures the market would be open for at least one day in that time range.
        # we then take the value for the last date i.e. 31 December if present else 30 December and so on.
        adjusted_start_date = year_closing_date - datetime.timedelta(days=10)
        # end in yfinance is not inclusive. hence 1 day is added to it to get the data for that day as well.
        stock_data = yfinance.download([self.ticker], start=adjusted_start_date, end=year_closing_date + datetime.timedelta(days=1))
        # ensure the data is sorted by date
        stock_data.sort_index(inplace=True)
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
        stocks_released: typing.List[stock.StockReleasedRecord],
        stocks_sold: typing.List[stock.StockSoldRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> None:
        super().__init__()
        self.entries = self.__get_entries(
            stocks_released=stocks_released,
            stocks_sold=stocks_sold,
            sbi_reference_rates=sbi_reference_rates,
            financial_year=financial_year
        )
    
    def __get_entries(
        self,
        stocks_released: typing.List[stock.StockReleasedRecord],
        stocks_sold: typing.List[stock.StockSoldRecord],
        sbi_reference_rates: typing.Dict[datetime.date, forex.SBIReferenceRatesRecord],
        financial_year: typing.Tuple[datetime.date, datetime.date],
    ) -> typing.List[ScheduleFAA3Record]:
        entries = []
        # for schedule FA, the timeframe to consider is 1 January to 31 December.
        year_start_date = datetime.date(year=financial_year[0].year, month=1, day=1)
        year_closing_date = datetime.date(year=financial_year[0].year, month=12, day=31)
        for stock_released in stocks_released:
            # ignore stocks release after year_closing_date since it is outisde the timeframe we are interested in.
            if stock_released.release_date > year_closing_date:
                logger.info(f'Skipping stock released on ${stock_released.release_date} for award number ${stock_released.award_number} on ${stock_released.broker} since it has occured after ${year_closing_date}')
                continue
            entries.append(ScheduleFAA3Record(
                stock=stock_released,
                sbi_reference_rates=sbi_reference_rates,
                financial_year=financial_year,
            ))
        for stock_sold in stocks_sold:
            # ignore stocks sold before year_start_date since it is outisde the timeframe we are interested in.
            # should not compare the release date for these stocks with year_end_date because the stocks can be bought at any time in the past.
            if stock_sold.sale_date < year_start_date:
                logger.info(f'Skipping stock sold on ${stock_sold.sale_date} for award number ${stock_sold.award_number} on ${stock_sold.broker} since it has occured before ${year_start_date}')
                continue
            entries.append(ScheduleFAA3Record(
                stock=stock_sold,
                sbi_reference_rates=sbi_reference_rates,
                financial_year=financial_year,
            ))
        return entries
