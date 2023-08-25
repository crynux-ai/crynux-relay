package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Hello(ctx *gin.Context) {
	helloHtml := `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8">
		<title>Crynux Hydrogen(H) Network Relay</title>
		<link rel="shortcut icon" type="image/x-icon" href="/static/favicon.ico">
		<style>
			* {
				font-family: "Segoe UI",SegoeUI,"Helvetica Neue",Helvetica,Arial,sans-serif
			}
			body {
				position: relative;
				margin: 0;
				padding: 0;
				width: 100%;
				height: 100%;
			}
			.card {
				position: absolute;
				width: 800px;
				height: 400px;
				left: 50%;
				top: 50%;
				margin-left: -400px;
				margin-top: -200px;
			}
			
			.title {
				font-size: 34px;
				font-weight: 600;
				text-align: center;
			}
			
			.link {
				margin-top: 36px;
				text-align: center;
			}
			
			.link a {
				text-decoration: none;
				color: #333;
			}
			
			.link a:hover {
				text-decoration: underline;
				color: #09A9FD;
			}
			
			.logo {
				margin-top: 120px;
			}
			
			.logo a {
				text-decoration: none;
			}
			
			.logo img {
				display: block;
				width: 200px;
				margin: 16px auto;
				border: 0;
				outline: 0;
			}
		</style>
	</head>
	<body>
		<div class="card">
			<div class="title">Relay Server for the Crynux Hydrogen(H) Network</div>
			<div class="link">
				<a href="/static/api_docs.html" target="_blank">API</a>
				&nbsp;|&nbsp;
				<a href="https://github.com/crynux-ai/h-relay" target="_blank">GitHub</a>
				&nbsp;|&nbsp;
				<a href="https://docs.crynux.ai" target="_blank">Documentation</a>
				&nbsp;|&nbsp;
				<a href="https://blog.crynux.ai" target="_blank">Blog</a>
			</div>
			<div class="logo">
				<a href="https://crynux.ai" target="_blank">
					<img src="/static/crynux.png" />
				</a>
			</div>
		</div>
	</body>
</html>
`
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(helloHtml))
}
