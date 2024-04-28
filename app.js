import express from 'express';
import { JSDOM } from 'jsdom';
import { Readability } from '@mozilla/readability';
import { engine } from 'express-handlebars';

const app = express();

app.engine('handlebars', engine());
app.set('view engine', 'handlebars');
app.set('views', './views');

const getUrl = request => {
    const urlParam = typeof request.query?.url === 'string' ? request.query?.url.trim() : '',
        titleParam = typeof request.query?.title === 'string' ? request.query?.title : '',
        textParam = typeof request.query?.text === 'string' ? request.query?.text : '';

    let url = urlParam ||
        (textParam.match(/\bhttps?:\/\/\S+/gi) || [])[0] ||
        (titleParam.match(/\bhttps?:\/\/\S+/gi) || [])[0] || 
        '';

    if(url && !url.startsWith("http://") && !url.startsWith("https://")) {
        url = `https://${url}`;
    }

    return url || null;
}

// alternative https://github.com/postlight/parser/issues
app.get("/", async (request, response) => {
    const url = getUrl(request);

    if(url) {

        let dom;

        try {
            dom = new JSDOM(
                await (await fetch(url)).text(),
                { url: url }
            );
        } catch(error) {
            return response.render(
                'error',
                {
                    url: url,
                    error: error
                }
            );
        }

        const article = new Readability(
            dom.window.document
        ).parse();

        dom = new JSDOM(article.content, { url: url});

        dom.window.document.querySelectorAll('a')
            .forEach(link => link.href = `/?url=${encodeURIComponent(link.href)}`);

        response.render(
            'article',
            {
                article: dom.window.document.body.innerHTML,
                title: article.title,
                url: url
            }
        );
    } else {
        response.render('home');
    }
});

app.listen(8080);