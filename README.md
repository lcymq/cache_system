# go实现缓存系统

### 缓存原理
- CPU中有一级缓存，二级缓存，三级缓存
- 缓存的工作原理：CPU需要读取数据时，会首先从缓存中查找需要的数据，如果找到了就直接处理，如果没有找到，就从内存中读取数据。
- 由于CPU中的缓存工作速度比内存还要快，所以缓存的使用能加快CPU处理速度。


### 应用场景
- 缓存不仅仅存在于硬件中，在各种软件系统中也处处可见。
  - web系统中，缓存存在于服务器端，客户端或者代理服务器中。
  - 广泛使用的 CDN 网络，也可以看作一个巨大的缓存系统。
  - 缓存在 Web 系统中的使用有很多好处，不仅可以减少网络流量，降低客户访问延迟，还可以减轻服务器负载。


### 缓存系统有以下功能
- 缓存数据的存储
- 过期数据项管理
- 内存数据导出，导入
- 提供CRUD接口