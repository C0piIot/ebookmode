import express from 'express';
import { JSDOM } from 'jsdom';
import { Readability } from '@mozilla/readability';// alternative https://github.com/postlight/parser/issues
import { engine } from 'express-handlebars';
import morgan from 'morgan';

const app = express();

app.engine('handlebars', engine());
app.use(morgan('combined'));
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

app.get("/", async (request, response) => {
    const baseContext = {
        build: process.env.BUILD_VERSION,
        url: getUrl(request),
        host: request.hostname
    }

    if(baseContext.url) {

        try {
            const documentResponse = await fetch(baseContext.url);
            const contentType = documentResponse.headers.get('Content-type');
            if (!contentType || contentType.indexOf("text/html") === -1) {
                throw new Error(`Invalid content type "${contentType}"`);
            }
            
            const originalDom = new JSDOM(
                await documentResponse.text(),
                baseContext
            );

            const article = new Readability(
                originalDom.window.document
            ).parse();

            if(!article){
                throw new Error("Error processing document html");
            }

            const cleanedDom = new JSDOM(article.content, { url: baseContext.url});
            cleanedDom.window.document.querySelectorAll('a')
                .forEach(link => {
                    link.href = `/?url=${encodeURIComponent(link.href)}`;
                    link.rel = 'nofollow';
                });

            response.render(
                'article',
                {
                    ...baseContext,
                    ...{
                        article: cleanedDom.window.document.body.innerHTML,
                        title: article.title,
                        urlEncoded: encodeURIComponent(baseContext.url),
                        excerpt: article.excerpt
                    }
                }
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
    } else {
        response.render('home', baseContext);
    }
});

app.listen(8080);
