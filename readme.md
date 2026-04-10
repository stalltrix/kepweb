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

<br>

## 效果展示

![demo](img/demo.jpg)

---

## 提示:

此主题为单用户设计主题，如果需要多用户使用。可以参考[kepweb-multi](https://github.com/stalltrix/kepweb-multi)项目。

kepweb-multi与kepweb的后端接口API是相同的，理论上两者的前端主题（即：**ui.html**）是可以互相替换的（更改一下url重写路径就行）。

<br>

单用户主题资源占用低，开发进度快。如无特殊需求，使用单用户主题即可。