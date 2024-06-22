import csv
import datetime
import logging
import urllib.request
import typing

from . import utils


logger = logging.getLogger(__name__)


class SBIReferenceRatesRecord:
    def __init__(self,
        date: datetime.date,
        source_currency: str,
        target_currency: str,
        tt_buy_exchange_rate: float,
    ) -> None:
        self.date = date
        self.source_currency = source_currency
        self.target_currency = target_currency
        self.tt_buy_exchange_rate = tt_buy_exchange_rate
    
    def __repr__(self) -> str:
        attrs = ', '.join(f'{key}={value!r}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'
    
    def __str__(self) -> str:
        attrs = ', '.join(f'{key}={value}' for key, value in vars(self).items())
        return f'{self.__class__.__name__}({attrs})'


class SBIReferenceRates:
    def __init__(self, file) -> None:
        if file:
            self.reference_rates_usd = self.__parse(csv.reader(file))
        else:
            self.reference_rates_usd = self.__fetch_and_parse()
    
    def __parse(self, reader) -> typing.Dict[datetime.date, SBIReferenceRatesRecord]:
        records = {}
        title_found = False
        for row in reader:
            if not title_found:
                title_found = True
                continue
            # date is at index 0
            # tt_buy is at index 2
            date = datetime.datetime.strptime(row[0], '%Y-%m-%d %H:%M').date()
            tt_buy_exchange_rate=float(row[2])
            # there are some edge cases where the TT BUY rate is 0. discard them.
            if tt_buy_exchange_rate == 0:
                logger.info('Discarding SBI TT BUY for %s because it is %s', date, tt_buy_exchange_rate)
                continue
            records[date] = SBIReferenceRatesRecord(
                date=date,
                source_currency="usd",
                target_currency="inr",
                tt_buy_exchange_rate=tt_buy_exchange_rate,
            )
        return records
    
    def __fetch_and_parse(self) -> typing.Dict[datetime.date, SBIReferenceRatesRecord]:
        url = 'https://raw.githubusercontent.com/sahilgupta/sbi-fx-ratekeeper/main/csv_files/SBI_REFERENCE_RATES_USD.csv'
        with urllib.request.urlopen(url=url) as response:
            lines = (line for line in response.read().decode('utf-8').splitlines())
        return self.__parse(csv.reader(lines))
        
    
    def on_date(self, date: datetime.date, adjust_to_last_day_of_previous_month: bool) -> typing.Tuple[datetime.date, SBIReferenceRatesRecord]:
        adjusted_date = date
        if adjust_to_last_day_of_previous_month:
            adjusted_date = utils.get_last_day_of_previous_moth(date)
        adjusted_for_missing_date = False
        while True:
            if adjusted_date in self.reference_rates_usd:
                break
            adjusted_for_missing_date = True
            adjusted_date = adjusted_date - datetime.timedelta(days=1)
        # if no record is found, it will error out
        return adjusted_date, self.reference_rates_usd[adjusted_date]
