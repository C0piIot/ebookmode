import express from 'express';
import { JSDOM } from 'jsdom';
import { Readability } from '@mozilla/readability';
import { engine } from 'express-handlebars';

const app = express();

app.engine('handlebars', engine());
app.set('view engine', 'handlebars');
app.set('views', './views');


app.get("/", async (request, response) => {
    const url = request.query?.url;
    if(url) {
        const doc = new JSDOM(
            await (await fetch(url)).text(),
            { url: url }
        );
        const reader = new Readability(doc.window.document);
        response.send(reader.parse().content);
    } else {
        response.render('home');
    }
});  

app.listen(80);