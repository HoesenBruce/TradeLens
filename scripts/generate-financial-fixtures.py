#!/usr/bin/env python3
"""Build public financial samples from invented constants; never read broker exports.
Run from any directory. --check verifies checked-in outputs byte for byte.
"""
import argparse
import csv
import html
import importlib.util
import io
import json
import random
import zipfile
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUTPUTS = {}


def put(path, data):
    OUTPUTS[path] = data.encode('utf-8') if isinstance(data, str) else data


def table(path, headers, rows, preamble=''):
    out = io.StringIO(newline='')
    out.write(preamble)
    writer = csv.writer(out, lineterminator='\n')
    writer.writerow(headers)
    writer.writerows(rows)
    put(path, out.getvalue())


def statement(path, headers, rows, section):
    # Only report grammar is retained; all account/fill values are invented here.
    grid = [['Synthetic Broker — generated fixture'], [section], headers, *rows]
    text = '<!DOCTYPE html><html><head><meta charset="utf-8"><title>Synthetic statement</title></head><body><table>'
    for row in grid:
        text += '<tr>' + ''.join('<td>' + html.escape(str(v)).replace('\u00a0', '&nbsp;') + '</td>' for v in row) + '</tr>\n'
    put(path, text + '</table></body></html>\n')
    return grid


def workbook(grid):
    # Minimal OOXML, deterministic ZIP headers, no author/device metadata.
    ns = 'http://schemas.openxmlformats.org/spreadsheetml/2006/main'
    rows = []
    for r, row in enumerate(grid, 1):
        cells = []
        for c, value in enumerate(row):
            ref = f'{chr(65 + c)}{r}'
            if isinstance(value, (int, float)):
                cells.append(f'<c r="{ref}" s="1"><v>{value}</v></c>')
            else:
                cells.append(f'<c r="{ref}" t="inlineStr"><is><t>{html.escape(str(value))}</t></is></c>')
        rows.append(f'<row r="{r}">' + ''.join(cells) + '</row>')
    parts = {
        '[Content_Types].xml': '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>',
        '_rels/.rels': '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>',
        'xl/workbook.xml': f'<workbook xmlns="{ns}" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Generated" sheetId="1" r:id="rId1"/></sheets></workbook>',
        'xl/_rels/workbook.xml.rels': '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>',
        'xl/styles.xml': f'<styleSheet xmlns="{ns}"><fonts count="1"><font/></fonts><fills count="1"><fill/></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs><cellXfs count="2"><xf/><xf numFmtId="22" applyNumberFormat="1"/></cellXfs></styleSheet>',
        'xl/worksheets/sheet1.xml': f'<worksheet xmlns="{ns}"><sheetData>' + ''.join(rows) + '</sheetData></worksheet>',
    }
    out = io.BytesIO()
    with zipfile.ZipFile(out, 'w', zipfile.ZIP_DEFLATED) as z:
        for name, text in parts.items():
            z.writestr(zipfile.ZipInfo(name, (2000, 1, 1, 0, 0, 0)), text)
    return out.getvalue()


SBI = 'api/internal/importer/testdata/'
headers = ['約定日', '銘柄', '銘柄コード', '市場', '取引', '約定数量', '約定単価', '手数料/諸経費等', '受渡金額/決済損益', '平均取得価額']
# Date-only order, duplicate rows, invalid qty, long/short and genbiki.
rows = []
for day, name, code, action, qty, price, fee, settlement in [
    (3, '合成甲', '9101', '株式現物買', 37, 730, '--', -27010),
    (4, '合成甲', '9101', '株式現物売', 37, 760, 7, 28113),
    (5, '合成乙', '731A', '信用新規買', 13, 420, 3, 0),
    (6, '合成乙', '731A', '信用返済売', 13, 440, 3, 254),
    (7, '合成丙', '732A', '信用新規売', 17, 650, 4, 0),
    (10, '合成丙', '732A', '信用返済買', 17, 620, 4, 502),
    (11, '合成丁', '733A', '株式現物買', 23, 370, '--', -8510),
    (11, '合成丁', '733A', '株式現物買', 23, 370, '--', -8510),
    (12, '合成戊', '9102', '信用新規買', 31, 810, '--', 0),
    (12, '合成不正', '9103', '株式現物買', '不正', 230, '--', 0),
    (13, '合成戊', '9102', '現引', 31, 830, 11, -25741),
]:
    rows.append([f'2031/02/{day:02}', name, code, '東証', action, qty, price, fee, settlement, ''])
table(SBI + 'sbi-trade-executions.csv', headers, rows, '約定履歴照会\n\n')
rows = []
for day, action, qty, price, pnl, basis in [
    (3, '株式現物買', 37, 730, -27010, ''), (3, '信用新規買', 13, 730, '--', ''),
    (3, '信用新規売', 17, 730, '--', ''), (4, '現物売', 37, 760, 28120, 730),
    (4, '信用返済売', 13, 760, 390, 730), (4, '信用返済買', 17, 710, 340, 730),
]:
    rows.append([f'2031/02/{day:02}', '合成銘柄', '733A', '東証', action, qty, price, '--', pnl, basis])
table(SBI + 'sbi-mixed-positions.csv', headers, rows, '約定履歴照会\n')
table(SBI + 'sbi-cash-transactions.csv', ['入出金日', '取引', '区分', '摘要', '出金額', '入金額'], [
    ['2031/02/03', '入金', '金融機関からの入金', '合成銀行', 0, 87000],
    ['2031/02/04', '出金', '金融機関への出金', '合成銀行', 23000, 0],
    ['2031/02/05', '入金', '利金・配当金', '合成配当', 0, 730],
    ['2031/02/06', '出金', 'その他', '合成調整', 41, 0],
    ['bad-date', '入金', 'その他', '合成不正', 0, 17],
], '\ufeff\n円貨入出金明細\n\n')
table(SBI + 'sbi-margin-generated-report.csv', ['約定日', '口座', '銘柄名', '取引', '数量', '売却/決済額', '単価', '平均取得価額', '実現損益(税引前・円)'], [
    ['2031/2/06', '特定', '合成甲 9101', '返済売', 37, 28971.5, 783.5, 751, '+1,183.50'],
    ['2031/2/07', '特定', '合成乙 9102', '返済買', 23, 16502.5, 717.5, 680, '-879.50'],
    ['2031/2/07', '特定', '合成現物 9103', '現物売', 11, 2530, 230, 220, 110],
    ['2031/2/07', '特定', 'コード不明', '返済売', 11, 2530, 230, 220, 110],
], '国内株式\n\n商品,実現損益(税引前・円),利益金額(円),損失金額(円)\n信用,0,0,0\n\n')

mt5_headers = ['Time', 'Deal', 'Symbol', 'Type', 'Direction', 'Volume', 'Price', 'Order', 'Commission', 'Fee', 'Swap', 'Profit', 'Balance', 'Comment']
mt5_rows = [
    ['2031.02.03 09:00:00', '810000', '', 'balance', '', '', '', '', 0, 0, 0, 17000, 17000, 'Generated deposit'],
    ['2031.02.04 11:20:00', '810001', 'AUDUSD', 'buy', 'in', '0.37', '0.71234', '820001', '-1.11', 0, 0, 0, 16998.89, ''],
    ['2031.02.04 15:35:30', '810002', 'AUDUSD', 'sell', 'out', '0.37', '0.71434', '820002', '-1.11', 0, '-0.73', 74, 17071.05, ''],
    ['2031.02.05 04:15:00', '810003', 'XAUUSD.m', 'buy', 'in', '0.07', '1\u00a0873.50', '820003', '-0.41', 0, 0, 0, 17070.64, ''],
    ['2031.02.05 22:30:10', '810004', 'XAUUSD.m', 'sell', 'out', 'bad', '1881.50', '820004', '-0.41', 0, '-0.17', 56, 17126.06, ''],
    ['', '', '', '', '', '', '', '', '-3.04', 0, '-0.90', 130, 17126.06, ''],
]
statement(SBI + 'mt5-statement.html', mt5_headers, mt5_rows, 'Deals')
# One numeric Excel date cell and one string timestamp retain both parser paths.
serial = (datetime(2031, 2, 4, 9, 20) - datetime(1899, 12, 30)).total_seconds() / 86400
xlsx_rows = [list(mt5_rows[1]), list(mt5_rows[2])]
xlsx_rows[0][0] = serial
put(SBI + 'mt5-statement.xlsx', workbook([['Synthetic Trade History Report'], ['Deals'], mt5_headers, *xlsx_rows]))
mt4_headers = ['Ticket', 'Open Time', 'Type', 'Size', 'Item', 'Price', 'S / L', 'T / P', 'Close Time', 'Price', 'Commission', 'Taxes', 'Swap', 'Profit']
statement(SBI + 'mt4-statement.html', mt4_headers, [
    ['830001', '2031.02.03 10:00:00', 'balance', '', '', '', '', '', '', '', '', '', '', 8700],
    ['830002', '2031.02.04 10:25:00', 'buy', '0.13', 'audusd', '0.71234', 0, 0, '2031.02.05 17:40:00', '0.71534', '-1.31', 0, '-0.47', 39],
    ['830003', '2031.02.06 15:30:00', 'sell', '0.07', 'eurjpy', '161.730', 0, 0, '2031.02.06 16:15:00', '161.530', '-0.71', 0, 0, 8.67],
    ['830004', '2031.02.07 12:10:00', 'sell limit', '0.11', 'audusd', '0.72000', 0, 0, '2031.02.07 16:10:00', 'cancelled', 0, 0, 0, 0],
    ['830005', '2031.02.10 09:55:00', 'buy', '0.17', 'audusd', 'broken', 0, 0, '2031.02.10 13:20:00', '0.71800', '-0.91', 0, 0, 17],
    ['Closed P/L:'], ['Open Trades:'],
], 'Closed Transactions:')

spec = importlib.util.spec_from_file_location('seed_demo', ROOT / 'scripts/seed-demo.py')
seed = importlib.util.module_from_spec(spec)
spec.loader.exec_module(seed)
book = seed.build_book(random.Random(210), datetime(2031, 2, 28, tzinfo=timezone.utc))


def export_trade(t):
    side = 'buy' if t['long'] else 'sell'
    fills = [dict(symbol=t['symbol'], instrument_type=t['instrument'], side=s,
                  quantity=t['qty'], price=p, fees=t['fee'], commission=0,
                  multiplier=t['mult'], executed_at=at.isoformat().replace('+00:00', 'Z'))
             for s, p, at in [(side, t['entry'], t['opened']), ('sell' if t['long'] else 'buy', t['exit'], t['closed'])]]
    gross = round((t['exit'] - t['entry']) * (1 if t['long'] else -1) * t['qty'] * t['mult'], 2)
    return dict(symbol=t['symbol'], instrument_type=t['instrument'], direction='long' if t['long'] else 'short',
                status='closed', opened_at=fills[0]['executed_at'], closed_at=fills[1]['executed_at'],
                qty_opened=t['qty'], qty_remaining=0, avg_entry_price=t['entry'], avg_exit_price=t['exit'],
                gross_pnl=gross, fees_total=round(t['fee'] * 2, 2), net_pnl=round(gross - t['fee'] * 2, 2),
                pnl_currency='USD', notes='Generated synthetic example; no real account records.', fills=fills)

trades = [export_trade(t) for t in book]
for i, t in enumerate(trades):
    t['setup'] = dict(name=seed.SETUPS[i % len(seed.SETUPS)])
    t['tags'] = [dict(name='Generated example', color='#38bdf8', kind='custom')]
# Retain open/partial-close JSON import cases using new quantities/prices.
open_trade = export_trade(dict(symbol='SYNTH', instrument='stock', long=True, qty=19, mult=1,
                              entry=73, exit=79, fee=1, opened=datetime(2031, 2, 27, 14, tzinfo=timezone.utc),
                              closed=datetime(2031, 2, 27, 15, tzinfo=timezone.utc)))
open_trade.update(status='open', closed_at=None, qty_remaining=12, avg_exit_price=79,
                  gross_pnl=42, fees_total=2, net_pnl=40)
open_trade['fills'][1]['quantity'] = 7
trades.append(open_trade)
put('docs/demo/tradermemos-demo-trades.json', json.dumps(dict(format_version=1, exported_at='2031-03-01T00:00:00Z',
    setups=[dict(name=n, description='Synthetic demo setup') for n in seed.SETUPS], trade_count=len(trades), trades=trades), indent=2) + '\n')
sample = export_trade(dict(symbol='SYNTH', instrument='stock', long=True, qty=19, mult=1,
                          entry=73, exit=79, fee=1, opened=datetime(2031, 2, 4, 14, tzinfo=timezone.utc),
                          closed=datetime(2031, 2, 4, 15, tzinfo=timezone.utc)))
put('web/public/sample-json-import.json', json.dumps(dict(format_version=1, exported_at='2031-02-04T16:00:00Z', trade_count=1, trades=[sample]), indent=2) + '\n')
fill_headers = ['Symbol', 'B/S', 'Qty', 'Fill Price', 'Trade Date', 'Commission']
sample_rows = [['SYNTH', 'BUY', 19, 73, '2031-02-04T14:00:00Z', 1], ['SYNTH', 'SELL', 19, 79, '2031-02-04T15:00:00Z', 1]]
table('web/public/sample-fill-import.csv', fill_headers, sample_rows)
table('api/testdata/generic_sample.csv', fill_headers, sample_rows + [['INVALID', 'BUY', 'notanumber', 17, '2031-02-04T14:00:00Z', 0]])

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    for path, expected in OUTPUTS.items():
        target = ROOT / path
        if args.check:
            assert target.read_bytes() == expected, f'{path}: regenerate fixtures'
        else:
            target.write_bytes(expected)
    print(f'{len(OUTPUTS)} independently generated fixtures {"verified" if args.check else "written"}')
