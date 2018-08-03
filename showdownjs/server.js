var markdownToXml = require("./olxutils");

var showdown = require('showdown'),
    showdownOptions = showdown.getDefaultOptions(false);

var express = require('express'),
    bodyParser = require('body-parser');

// load showdown default options
for (var opt in showdownOptions) {
    if (showdownOptions.hasOwnProperty(opt)) {
        if (showdownOptions[opt].defaultValue === false) {
            showdownOptions[opt].default = null;
        } else {
            showdownOptions[opt].default = showdownOptions[opt].defaultValue;
        }
    }
}

function parseSDOptions(flavor) {
    var options = {},
        flavorOpts = showdown.getFlavorOptions(flavor) || {};

    // if flavor is not undefined, let's tell the user we're loading that preset
    if (flavor) {
        console.info('Loading ' + flavor + ' flavor.');
    }

    return options;
}

var sdConverter = new showdown.Converter(parseSDOptions('github'));

var app = express();

app.use(bodyParser.json())
app.use(bodyParser.urlencoded({extended: true}))

// Which port to listen on
app.set('port', 6222);

app.post('/makeolx', function(req, res, next) {
    var { content } = req.body;
    try {
        var x = markdownToXml(content);
        res.send({content: x})
    } catch (err) {
        next(err);
    }
});

app.post('/makemarkdown', function(req, res, next) {
    var { content } = req.body;
    try {
        var x = sdConverter.makeMarkdown(content);
        res.send({content: x})
    } catch (err) {
        next(err);
    }
});

app.post('/makehtml', function(req, res, next) {
    var { content } = req.body;
    try {
        var x = sdConverter.makeHtml(content);
        res.send({content: x})
    } catch (err) {
        next(err);
    }
});

// Start listening for HTTP requests
var server = app.listen(app.get('port'), function() {
    var host = server.address().address;
    var port = server.address().port;

    console.log('ShowdownJS API listening at http://%s:%s', host, port);
})
