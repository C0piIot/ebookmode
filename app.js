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
    const baseContext = {
        build: process.env.BUILD_VERSION,
        url: getUrl(request)
    }
console.log(baseContext)
    if(baseContext.url) {

        let dom;

        try {
            dom = new JSDOM(
                await (await fetch(baseContext.url)).text(),
                baseContext
            );
        } catch(error) {
            return response.render(
                'error',
                {
                    ...baseContext,
                    ...{ error: error }
                }
            );
        }

        const article = new Readability(
            dom.window.document
        ).parse();

        dom = new JSDOM(article.content, { url: baseContext.url});

        dom.window.document.querySelectorAll('a')
            .forEach(link => {
                link.href = `/?url=${encodeURIComponent(link.href)}`;
                link.rel = 'nofollow';
            });

        response.render(
            'article',
            {
                ...baseContext,
                ...{
                    article: dom.window.document.body.innerHTML,
                    title: article.title,
                    urlEncoded: encodeURIComponent(baseContext.url),
                    excerpt: article.excerpt,
                    host: request.hostname
                }
            }
        );
    } else {
        response.render('home');
    }
});

app.listen(8080);