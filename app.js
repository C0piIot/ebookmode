import express from 'express';
import { JSDOM } from 'jsdom';
import { Readability } from '@mozilla/readability';
import { engine } from 'express-handlebars';

const app = express();

app.engine('handlebars', engine());
app.set('view engine', 'handlebars');
app.set('views', './views');

const getUrl = request => {
    let url = request.query?.url?.trim() ||
        (request.query?.text?.match(/\bhttps?:\/\/\S+/gi) || [])[0] ||
        (request.query?.title?.match(/\bhttps?:\/\/\S+/gi) || [])[0];
    
    if(!url) {
        return null;
    }

    if(!url.startsWith("http://") && !url.startsWith("https://")) {
        url = `https://${url}`;
    }

    return url;
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