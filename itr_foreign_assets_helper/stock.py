import typing

from datetime import date

class Stock:
    def __init__(
        self,
        source_metadata: typing.Dict,
        platform: str,
        comments: str,
        award_number: int,
    ) -> None:
        self.source_metadata = source_metadata
        self.platform = platform
        self.comments = comments
        self.award_number = award_number
    
    def __repr__(self) -> str:
        attrs = ', '.join(f'{key}={value!r}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'
    
    def __str__(self) -> str:
        attrs = ', '.join(f'{key}={value}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'

class StockIssued(Stock):
    def __init__(
        self,
        source_metadata: typing.Dict,
        platform: str,
        comments: str,
        award_number: int,
        shares_issued: int,
        release_date: date,
        market_value_per_share: float,
    ) -> None:
        super().__init__(source_metadata, platform, comments, award_number)
        self.transaction_type = "issued"
        self.shares_issued = shares_issued
        self.release_date = release_date
        self.market_value_per_share = market_value_per_share


class StockSold(Stock):
    def __init__(
        self,
        source_metadata: typing.Dict,
        platform: str,
        comments: str,
        award_number: int,
        shares_sold: int,
        sale_date: date,
        sale_value_per_share: float,
    ) -> None:
        super().__init__(source_metadata, platform, comments, award_number)
        self.transaction_type = "sold"
        self.shares_sold = shares_sold
        self.sale_date = sale_date
        self.sale_value_per_share = sale_value_per_share
