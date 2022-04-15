import os

for i in range(3):
    os.system(
        f'wt go run D:\Projects\Go\RelayKV\server --http 127.0.0.1:9870,127.0.0.1:9871,127.0.0.1:9872 --cluster 127.0.0.1:7651,127.0.0.1:7652,127.0.0.1:7653 --id {i}')
