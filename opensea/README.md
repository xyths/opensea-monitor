# OpenSea 说明

## 限速问题

API Key申请中。

## 更新TopN NFT列表

由于OpenSea API没有获取头部Collection的接口，因此离线整理csv文档，手工导入MongoDB进行更新。

```shell
mongoimport -d opensea -c projects --drop --type=csv --headerline data/top100.csv
```

### 其他问题

`collections`接口`limit`最大是300。