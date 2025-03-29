from allgrpc import server
from milvus import Milvus
from File import Data

def store():
    f = Data()
    m = Milvus()

    path = "/home/chenyun/下载/train1.json"
    data = f.readFile(path)
    print("开始存储")
    m.storejson(data)
    print("存储完成")


if __name__ == "__main__":
    m = Milvus()
    f = Data()
    server.server(m,f)