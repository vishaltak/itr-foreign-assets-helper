import argparse
import logging

from . import etrade

logging.basicConfig(
    level=logging.DEBUG,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    handlers=[
        logging.FileHandler(".log.txt"),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

def main():
    parser = argparse.ArgumentParser(
        prog='itr-foreign-assets-helper',
        description='itr-foreign-assets-helper',
    )
    parser.add_argument('--etrade-holdings',
        type=argparse.FileType('rb'),
        required=True,
        help='ETrade holdings file.')
    args = parser.parse_args()

    etrade_data = etrade.ETrade()
    etrade_data.get_shares_issued(holdings_file=args.etrade_holdings)
    
    logger.info('ITR Data')
    logger.info('ETrade')
    logger.info(etrade_data)
