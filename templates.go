package main

import (
	"encoding/xml"
)

type title []byte
type a struct {
	Href   string `xml:"href,attr"`
	Title3 string `xml:",chardata"`
}
type li struct {
	//  Class string `xml:"class,attr"`
	A a `xml:"a"`
}
type listMenu struct {
	XMLName xml.Name `xml:"ul"`
	Lis     []li     `xml:"li"`
}
type mdContent struct {
	level                 string
	title3                title
	listMenu              listMenu
	ListMenu              string
	Content               string
	ContentStyle          string
	MenuStyle             string
	MenuWrapStyle         string
	ScrollBar             string
	MenuLogo, ContentLogo string
}

var templContent = `<!doctype html>
<html>
	<head>
		<meta charset="utf-8"/>
		<meta http-equiv="X-UA-Compatible" content="chrome=1"/>
        <link href="/-/assets/monokai-sublime.min.css" rel="stylesheet">
		<title></title>
	</head>
	<body>
      <div class="container">
        <div class="nav-wrap">
          <div class="markdown-body" style="float:left;">
            <div class="nav">{{.ListMenu}}</div>
            {{.MenuLogo}}
          </div>      
        </div>
      	<div class="markdown-body">
            {{.Content}}{{.ContentLogo}}
        </div>
      <div>
      <script src="/-/assets/jquery-3.2.1.min.js"></script>
      <script src="/-/assets/highlight.min.js"></script>
      <script src="/-/assets/highlight.pack.js"></script>
      <script>
        $(document).ready(function() {
          $('pre code').each(function(i, block) {
          	if(block.className!=""){
          		hljs.highlightBlock(block);
          	}else{
          		$(this).addClass("hljs");
          	}            
          });
        });
      </script>
	</body>
</html>
`
