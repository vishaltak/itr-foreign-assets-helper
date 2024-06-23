import datetime
import typing

from . import forex


import datetime

from . import forex


class Date:

    def __init__(
        self,
        type: str,
        date: datetime.date,
        sbi_reference_rates: forex.SBIReferenceRates,
    ) -> None:
        self.type = type
        self.actual_date = date
        adjust_to_last_day_of_previous_month = None
        if self.type == 'release_date':
            adjust_to_last_day_of_previous_month = True
        elif self.type == 'sale_date':
            adjust_to_last_day_of_previous_month = True
        elif self.type == 'peak_closing_high_date':
            adjust_to_last_day_of_previous_month = True
        elif self.type == 'year_closing_date':
            adjust_to_last_day_of_previous_month = False
        else:
            raise ValueError(f'Invalid value for event type: {self.type}')
        self.adjusted_date_for_sbi_reference_rate, self.sbi_reference_rate = sbi_reference_rates.on_date(
            self.actual_date,
            adjust_to_last_day_of_previous_month=adjust_to_last_day_of_previous_month
        )

    def __repr__(self) -> str:
        attrs = ', '.join(f'{key}={value!r}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'
    
    def __str__(self) -> str:
        attrs = ', '.join(f'{key}={value}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'


class CashRecord:

    def __init__(
        self,
        sbi_reference_rates: forex.SBIReferenceRates,
        source_metadata: typing.Dict,
        broker: str,
        comments: str,
        amount: float,
    ) -> None:
        self.sbi_reference_rates = sbi_reference_rates
        self.source_metadata = source_metadata
        self.broker = broker
        self.comments = comments
        self.amount = amount

    def __repr__(self) -> str:
        attrs = ', '.join(f'{key}={value!r}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'
    
    def __str__(self) -> str:
        attrs = ', '.join(f'{key}={value}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'


class ShareRecord:

    def __init__(
        self,
        sbi_reference_rates: forex.SBIReferenceRates,
        source_metadata: typing.Dict,
        broker: str,
        comments: str,
        ticker: str,
        award_number: int,
    ) -> None:
        self.sbi_reference_rates = sbi_reference_rates
        self.source_metadata = source_metadata
        self.broker = broker
        self.comments = comments
        self.ticker = ticker
        self.award_number = award_number
    
    @property
    def transaction_type(self):
        raise NotImplementedError(f'{__name__} should set transaction_type')

    def __repr__(self) -> str:
        attrs = ', '.join(f'{key}={value!r}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'
    
    def __str__(self) -> str:
        attrs = ', '.join(f'{key}={value}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'


class ShareReleasedRecord(ShareRecord):

    def __init__(
        self,
        sbi_reference_rates: forex.SBIReferenceRates,
        source_metadata: typing.Dict,
        broker: str,
        comments: str,
        ticker: str,
        award_number: int,
        shares_released: int,
        release_date: datetime.date,
        market_value_per_share: float,
    ) -> None:
        super().__init__(sbi_reference_rates, source_metadata, broker, comments, ticker, award_number)
        self.shares_released = shares_released
        self.release_date = Date(date=release_date, type='release_date', sbi_reference_rates=self.sbi_reference_rates)
        self.market_value_per_share = market_value_per_share
    
    @property
    def transaction_type(self):
        return 'released'


class ShareSoldRecord(ShareRecord):
    def __init__(
        self,
        sbi_reference_rates: forex.SBIReferenceRates,
        source_metadata: typing.Dict,
        broker: str,
        comments: str,
        ticker: str,
        award_number: int,
        release_date: datetime.date,
        market_value_per_share: float,
        shares_sold: int,
        sale_date: datetime.date,
        sale_value_per_share: float,
    ) -> None:
        super().__init__(sbi_reference_rates, source_metadata, broker, comments, ticker, award_number)
        self.release_date = Date(date=release_date, type='release_date', sbi_reference_rates=self.sbi_reference_rates)
        self.market_value_per_share = market_value_per_share
        self.shares_sold = shares_sold
        self.sale_date = Date(date=sale_date, type='sale_date', sbi_reference_rates=self.sbi_reference_rates)
        self.sale_value_per_share = sale_value_per_share
    
    @property
    def transaction_type(self):
        return 'sold'
