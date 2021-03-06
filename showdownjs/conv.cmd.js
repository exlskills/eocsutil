var markdownToXml = require("./olxutils");
var unescapeMd = require("./unescapemd");

var yargs = require('yargs'),
    fs = require('fs'),
    Messenger = require('./messenger.js'),
    showdown = require('showdown'),
    showdownOptions = showdown.getDefaultOptions(false);

yargs.reset()
    .usage('Usage: showdown makehtml [options]')
    .example('showdown makehtml -i', 'Reads from stdin and outputs to stdout')
    .example('showdown makehtml -i foo.md -o bar.html', 'Reads \'foo.md\' and writes to \'bar.html\'')
    .example('showdown makehtml -i --flavor="github"', 'Parses stdin using GFM style')
    .version()
    .alias('v', 'version')
    .config('c')
    .alias('c', 'config')
    .help('h')
    .alias('h', 'help')
    .option('i', {
        alias : 'input',
        describe: 'Input source file (md or html). If omitted or empty, reads from stdin',
        type: 'string'
    })
    .option('o', {
        alias : 'output',
        describe: 'Output target file (md or html). If omitted or empty, writes to stdout',
        type: 'string',
        default: false
    })
    .option('u', {
        alias : 'encoding',
        describe: 'Input encoding',
        type: 'string'
    })
    .option('a', {
        alias : 'append',
        describe: 'Append data to output instead of overwriting',
        type: 'string',
        default: false
    })
    .option('e', {
        alias : 'extensions',
        describe: 'Load the specified extensions. Should be valid paths to node compatible extensions',
        type: 'array'
    })
    .option('p', {
        alias : 'flavor',
        describe: 'Run with a predetermined flavor of options. Default is vanilla',
        type: 'string'
    })
    .option('q', {
        alias: 'quiet',
        description: 'Quiet mode. Only print errors',
        type: 'boolean',
        default: false
    })
    .option('m', {
        alias: 'mute',
        description: 'Mute mode. Does not print anything',
        type: 'boolean',
        default: false
    });

// load showdown default options
for (var opt in showdownOptions) {
    if (showdownOptions.hasOwnProperty(opt)) {
        if (showdownOptions[opt].defaultValue === false) {
            showdownOptions[opt].default = null;
        } else {
            showdownOptions[opt].default = showdownOptions[opt].defaultValue;
        }
        yargs.option(opt, showdownOptions[opt]);
    }
}

function run (makeOpt) {
    'use strict';
    var argv = yargs.argv,
        readMode = (!argv.i || argv.i === '') ? 'stdin' : 'file',
        writeMode = (!argv.o || argv.o === '') ? 'stdout' : 'file',
        msgMode = (writeMode === 'file') ? 'stdout' : 'stderr',
        /**
         * MSG object
         * @type {Messenger}
         */
        messenger = new Messenger(msgMode, argv.q, argv.m),
        read = (readMode === 'stdin') ? readFromStdIn : readFromFile,
        write = (writeMode === 'stdout') ? writeToStdOut : writeToFile,
        enc = argv.encoding || 'utf-8',
        flavor =  argv.p,
        append = argv.a || false,
        options = parseOptions(flavor),
        converter = new showdown.Converter(options),
        from, to;

    // Load extensions
    if (argv.e) {
        messenger.printMsg('Loading extensions');
        for (var i = 0; i < argv.e.length; ++i) {
            try {
                var ext = require(argv.e[i]);
                converter.addExtension(ext, argv.e[i]);
            } catch (e) {
                messenger.printError('Could not load extension ' + argv.e[i] + '. Reason:');
                messenger.errorExit(e);
            }
        }
    }

    messenger.printMsg('...');
    // read the input
    messenger.printMsg('Reading data from ' + readMode + '...');
    from = read(enc);

    // process the input
    messenger.printMsg('Parsing markdown...');
    if (makeOpt === 'makehtml') {
        to = converter.makeHtml(from);
    } else if (makeOpt === 'makemarkdown') {
        to = converter.makeMarkdown(from);
    } else if (makeOpt === 'makeolx') {
        to = markdownToXml(from);
    } else if (makeOpt === 'unescapemd') {
        to = unescapeMd(from)
    } else {
        messenger.printError('Invalid conversion type: ' + makeOpt + '. Must be either makehtml or makeolx or makemarkdown.');
        messenger.errorExit(e);
    }


    // write the output
    messenger.printMsg('Writing data to ' + writeMode + '...');
    write(to, append);
    messenger.okExit();

    function parseOptions (flavor) {
        var options = {},
            flavorOpts = showdown.getFlavorOptions(flavor) || {};

        // if flavor is not undefined, let's tell the user we're loading that preset
        if (flavor) {
            messenger.printMsg('Loading ' + flavor + ' flavor.');
        }

        for (var opt in argv) {
            if (argv.hasOwnProperty(opt)) {
                // first we load the default options
                if (showdownOptions.hasOwnProperty(opt) && showdownOptions[opt].default !== null) {
                    options[opt] = showdownOptions[opt].default;
                }

                // we now override defaults with flavor, if a flavor was indeed passed
                if (flavorOpts.hasOwnProperty(opt)) {
                    options[opt] = flavorOpts[opt];
                }

                // lastly we override with explicit passed options
                // being careful not to pass CLI specific options, such as -v, -h or --extensions
                if (showdownOptions.hasOwnProperty(opt)) {
                    if (argv[opt] === true) {
                        messenger.printMsg('Enabling option ' + opt);
                        options[opt] = argv[opt];
                    } else if (argv[opt] === false) {
                        options[opt] = argv[opt];
                    }
                }
            }
        }
        return options;
    }

    function readFromStdIn () {
        try {
            return fs.readFileSync('/dev/stdin').toString();
        } catch (e) {
            let err = new Error('Could not read from stdin, reason: ' + e.message);
            messenger.errorExit(err);
        }
    }

    function readFromFile (encoding) {
        try {
            return fs.readFileSync(argv.i, encoding);
        } catch (err) {
            messenger.errorExit(err);
        }
    }

    function writeToStdOut (html) {
        return process.stdout.write(html);
    }

    function writeToFile (html, append) {
        // If a flag is passed, it means we should append instead of overwriting.
        // Only works with files, obviously
        var write = (append) ? fs.appendFileSync : fs.writeFileSync;
        try {
            write(argv.o, html);
        } catch (err) {
            messenger.errorExit(err);
        }
    }
}

module.exports = exports = {
    run: run
};