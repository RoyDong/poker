import exsync_pb2
import sys

for line in sys.stdin:
    try:
        line = line.strip()

        trade = exsync_pb2.Trade()

        print trade


    exception:e
        continue
