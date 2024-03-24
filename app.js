import express from 'express';
import { JSDOM } from 'jsdom';

const app = express();

app.get("/", (request, response) => {
    const url = request.query?.url;
    if(url) {
        const doc = new JSDOM("<body>Look at this cat: <img src='./cat.jpg'></body>", {
            url: "https://www.example.com/the-page-i-got-the-source-from"
          });
        const reader = new Readability(doc.window.document);
        const article = reader.parse();
        response.send(reader.content);
    } else {
        response.send("url is missing");
    }
});  
app.listen(80);