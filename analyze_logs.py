#!/usr/bin/env python3
import sys
import re
import argparse
from datetime import datetime, timedelta
from collections import defaultdict
import json
import urllib.request

# Regex to parse log lines
# Example: 2026/02/20 19:37:47 SESSION IN 2:5083/85.1 OK R:0/0 S:0/0 Time:1ms
LOG_PATTERN = re.compile(
    r'^(?P<date>\d{4}/\d{2}/\d{2}) (?P<time>\d{2}:\d{2}:\d{2}) SESSION (?P<direction>\w+) (?P<address>\S+) (?P<status>\w+) R:(?P<r_files>\d+)/(?P<r_bytes>\d+) S:(?P<s_files>\d+)/(?P<s_bytes>\d+) Time:(?P<duration>\S+)$'
)

def parse_duration(duration_str):
    """Parses Go-style duration strings (e.g., 1ms, 2m0.007s, 0s) into seconds."""
    total_seconds = 0.0
    s = duration_str
    
    # Parse Hours
    if 'h' in s:
        parts = s.split('h', 1)
        total_seconds += float(parts[0]) * 3600
        s = parts[1]
    
    # Parse Minutes (ensure we don't confuse 'm' with 'ms')
    if 'm' in s and not s.endswith('ms') and not 'ms' in s.split('m')[0]: 
        parts = s.split('m', 1)
        total_seconds += float(parts[0]) * 60
        s = parts[1]
        
    # Parse Seconds (ensure we don't confuse 's' with 'ms', 'us', 'ns')
    if 's' in s and not s.endswith('ms') and not s.endswith('us') and not s.endswith('ns'):
        parts = s.split('s', 1)
        total_seconds += float(parts[0])
        s = parts[1]
        
    # Parse Milliseconds
    if 'ms' in s:
        parts = s.split('ms', 1)
        total_seconds += float(parts[0]) / 1000
        s = parts[1]
        
    return total_seconds

def format_bytes(size):
    power = 2**10
    n = 0
    power_labels = {0 : '', 1: 'K', 2: 'M', 3: 'G', 4: 'T'}
    while size > power:
        size /= power
        n += 1
    return f"{size:.2f} {power_labels[n]}B"

def get_todays_holiday(country_code):
    """Fetches today's holiday from the date.nager.at API."""
    today = datetime.now()
    api_url = f"https://date.nager.at/api/v3/PublicHolidays/{today.year}/{country_code}"
    try:
        # Use a short timeout to avoid delaying the script
        with urllib.request.urlopen(api_url, timeout=2) as response:
            if response.status == 200:
                holidays = json.loads(response.read())
                today_str = today.strftime('%Y-%m-%d')
                for holiday in holidays:
                    if holiday.get('date') == today_str:
                        return holiday.get('name')
    except Exception:
        # Silently fail if the API is unreachable or there's an error
        return None
    return None

def main():
    parser = argparse.ArgumentParser(description='Analyze momail session logs.')
    parser.add_argument('logfile', nargs='?', default='./run/logs/session.log', help='Path to session.log file')
    parser.add_argument('--hours', type=int, default=24, help='Analyze last N hours (default: 24)')
    # Add country argument for holiday check, default to Kazakhstan based on user context
    parser.add_argument('--country', type=str, help='Country code for holiday lookup (e.g., US, GB, KZ)')
    args = parser.parse_args()

    try:
        with open(args.logfile, 'r') as f:
            lines = f.readlines()
    except FileNotFoundError:
        print(f"Error: File '{args.logfile}' not found.")
        sys.exit(1)

    # --- Holiday Check ---
    if args.country:
        holiday_name = get_todays_holiday(args.country)
        if holiday_name:
            print(f"By the way, today is {holiday_name}!\n")

    parsed_entries = []
    for line in lines:
        match = LOG_PATTERN.match(line.strip())
        if match:
            data = match.groupdict()
            dt_str = f"{data['date']} {data['time']}"
            dt = datetime.strptime(dt_str, '%Y/%m/%d %H:%M:%S')
            
            entry = {
                'dt': dt,
                'direction': data['direction'],
                'address': data['address'],
                'status': data['status'],
                'r_files': int(data['r_files']),
                'r_bytes': int(data['r_bytes']),
                's_files': int(data['s_files']),
                's_bytes': int(data['s_bytes']),
                'duration': parse_duration(data['duration'])
            }
            parsed_entries.append(entry)

    if not parsed_entries:
        print("No valid session entries found in log.")
        sys.exit(0)

    # Determine the analysis window based on the last entry in the log
    last_entry_time = parsed_entries[-1]['dt']
    cutoff_time = last_entry_time - timedelta(hours=args.hours)

    filtered_entries = [e for e in parsed_entries if e['dt'] >= cutoff_time]

    if not filtered_entries:
        print(f"No entries found in the last {args.hours} hours (relative to last log entry: {last_entry_time}).")
        sys.exit(0)

    # --- Aggregation ---
    width = 72
    total_sessions = len(filtered_entries)
    by_direction = defaultdict(int)
    by_status = defaultdict(int)
    total_r_bytes = 0
    total_s_bytes = 0
    link_stats = defaultdict(lambda: {'count': 0, 'err': 0, 'r_bytes': 0, 's_bytes': 0, 'duration': 0})
    
    for e in filtered_entries:
        by_direction[e['direction']] += 1
        by_status[e['status']] += 1
        total_r_bytes += e['r_bytes']
        total_s_bytes += e['s_bytes']
        
        ls = link_stats[e['address']]
        ls['count'] += 1
        if e['status'] != 'OK':
            ls['err'] += 1
        ls['r_bytes'] += e['r_bytes']
        ls['s_bytes'] += e['s_bytes']
        ls['duration'] += e['duration']


    # --- Output ---
    print(f"\n Mailer's statistics")
    print(f" Totals for {args.hours} hours (ending {last_entry_time})")
    print("=" * width)
    print(f"Total Sessions: {total_sessions}")
    print(f"  Incoming:     {by_direction['IN']:<5}  Success: {by_status['OK']}")
    print(f"  Outgoing:     {by_direction['OUT']:<5}  Errors:  {by_status['ERR']}")
    print("-" * width)
    print(f"Traffic:")
    print(f"  Received:     {format_bytes(total_r_bytes)}")
    print(f"  Sent:         {format_bytes(total_s_bytes)}")
    print(f"\n Link summary:")
    print("=" * width)
    
    print(f"{'Address':<22} | {'Sess':<4} | {'Err':<3} | {'Recv':<10} | {'Sent':<10} | {'Avg Time':<8}")
    print("-" * width)
    for addr, stats in sorted(link_stats.items(), key=lambda x: x[1]['count'], reverse=True):
        avg_time = stats['duration'] / stats['count'] if stats['count'] > 0 else 0
        print(f"{addr:<22} | {stats['count']:<4} | {stats['err']:<3} | {format_bytes(stats['r_bytes']):<10} | {format_bytes(stats['s_bytes']):<10} | {avg_time:.2f}s")
    print("=" * width)

    print(f"\n Graph (X - Failed; # - Succeeeded)")
    print("=" * width)
    
    addr_width = 18
    graph_width = width - addr_width - 3
    
    start_time = cutoff_time
    end_time = last_entry_time
    duration_sec = (end_time - start_time).total_seconds()
    if duration_sec <= 0: duration_sec = 1
    
    sec_per_col = duration_sec / graph_width
    
    # Prepare data rows
    link_rows = defaultdict(lambda: [' '] * graph_width)
    
    for e in filtered_entries:
        addr = e['address']
        offset = (e['dt'] - start_time).total_seconds()
        col = int(offset / sec_per_col)
        if col < 0: col = 0
        if col >= graph_width: col = graph_width - 1
        
        char = '#' if e['status'] == 'OK' else 'X'
        if link_rows[addr][col] != 'X': # Error takes precedence
            link_rows[addr][col] = char

    # Build Ruler
    ruler = [' '] * graph_width
    t = start_time.replace(minute=0, second=0, microsecond=0)
    if t < start_time: t += timedelta(hours=1)
        
    duration_hours = (end_time - start_time).total_seconds() / 3600
    hour_step = 1
    if duration_hours > 12:
        hour_step = 4
    elif duration_hours > 6:
        hour_step = 2

    while t <= end_time:
        if t.hour % hour_step == 0:
            offset = (t - start_time).total_seconds()
            col = int(offset / sec_per_col)
            if 0 <= col < graph_width - 1:
                if ruler[col] == ' ' and ruler[col+1] == ' ':
                    h = t.strftime("%H")
                    ruler[col] = h[0]
                    ruler[col+1] = h[1]
        t += timedelta(hours=1)
        
    print(f"{'Address':<{addr_width}} | {''.join(ruler)}")
    print("-" * width)
    
    for addr, _ in sorted(link_stats.items(), key=lambda x: x[1]['count'], reverse=True):
        row = "".join(link_rows[addr])
        print(f"{addr:<{addr_width}} | {row}")
    
    print("-" * width)
    print(f"Range: {start_time.strftime('%Y-%m-%d %H:%M')} - {end_time.strftime('%H:%M')}")
    print("=" * width)
    print(f"\n")

if __name__ == "__main__":
    main()
