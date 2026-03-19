# kepweb

kep webUI 界面，取代默认的kepcli，实现发帖/回帖/修改 webUI化


配置方法：

neighbors指向自己的阶段，token填local_token
```json
"neighbors": [
		{
			"url": "http://127.0.0.1:8081",
			"token": "token0"
		}
	]
```


与kepcli的发送指令差不多
```bash
kepcli -act send -addr http://127.0.0.1:8081 -auth token0
```


kep实现有一个local_token，与普通token没用多大区别，唯一区别就是不会再把msg发回来，设计为local环境使用。