import datetime
import typing


class CashRecord:

    def __init__(
        self,
        source_metadata: typing.Dict,
        broker: str,
        comments: str,
        amount: float,
    ) -> None:
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
        source_metadata: typing.Dict,
        broker: str,
        comments: str,
        ticker: str,
        award_number: int,
    ) -> None:
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
        source_metadata: typing.Dict,
        broker: str,
        comments: str,
        ticker: str,
        award_number: int,
        shares_released: int,
        release_date: datetime.date,
        market_value_per_share: float,
    ) -> None:
        super().__init__(source_metadata, broker, comments, ticker, award_number)
        self.shares_released = shares_released
        self.release_date = release_date
        self.market_value_per_share = market_value_per_share
    
    @property
    def transaction_type(self):
        return 'released'


class ShareSoldRecord(ShareRecord):
    def __init__(
        self,
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
        super().__init__(source_metadata, broker, comments, ticker, award_number)
        self.release_date = release_date
        self.market_value_per_share = market_value_per_share
        self.shares_sold = shares_sold
        self.sale_date = sale_date
        self.sale_value_per_share = sale_value_per_share
    
    @property
    def transaction_type(self):
        return 'sold'
