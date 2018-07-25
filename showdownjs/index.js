const yargs = require('yargs');

yargs
    .version()
    .alias('v', 'version')
    .option('h', {
        alias: 'help',
        description: 'Show help'
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
    })
    .usage('Usage: showdown <command> [options]')
    .demand(1, 'You must provide a valid command')
    .command('makehtml', 'Converts markdown into html')
    .example('showdown makehtml -i foo.md -o bar.html', 'Converts \'foo.md\' to \'bar.html\'')
    .command('makemarkdown', 'Converts html into md')
    .example('showdown makemarkdown -i foo.html -o bar.md', 'Converts \'foo.html\' to \'bar.md\'')
    .wrap(yargs.terminalWidth());

const argv = yargs.argv,
    command = argv._[0];

if (command === 'makehtml') {
    require('./conv.cmd.js').run('makehtml');
} else if (command === 'makemarkdown') {
    require('./conv.cmd.js').run('makemarkdown');
} else {
    yargs.showHelp();
}

if (argv.help) {
    yargs.showHelp();
}

process.exit(0);
