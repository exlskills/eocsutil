module.exports = function(content) {
    return content.replace(/(?<=`.*)(&amp;)(?=.*`)/g, "&")
        .replace(/(?<=```.*)(&amp;)(?=.*```)/gms, "&")
        .replace(/(?<=`.*)(&pipe;)(?=.*`)/g, "|")
        .replace(/(?<=```.*)(&pipe;)(?=.*```)/gms, "|")
        .replace(/(?<=`.*)(&gt;)(?=.*`)/g, ">")
        .replace(/(?<=```.*)(&gt;)(?=.*```)/gms, ">")
        .replace(/(?<=`.*)(&lt;)(?=.*`)/g, "<")
        .replace(/(?<=```.*)(&lt;)(?=.*```)/gms, "<");
};
