'use strict';

var Parser = require('ansi-sgr-parser');

var parser = new Parser();

var display = {
  element: null,

  rootElement: document.getElementById('content'),

  getElement: function () {
    return this.element || this.rootElement;
  },

  push: function (part) {
    var element = this.getElement();

    if (typeof part === 'string') {
      element.appendChild(document.createTextNode(part));
    } else {
      var isReset = part.attrs.some(function (attr) {
        return attr === 'reset';
      });

      if (isReset) {
        this.element = null;
      } else {
        var span = document.createElement('span');
        part.attrs.forEach(function (attr) {
          if (!attr) return;
          span.classList.add(attr.replace(/ /g, '-'));
        });
        element.appendChild(span);

        this.element = span;
      }
    }
  }
};

var conn = new WebSocket('ws://' + location.host + '/ws');
conn.onmessage = function (e) {
  var message = JSON.parse(e.data);
  if (message.type === 'text') {
    var parts = parser.add(message.data);
    parts.forEach(function (part) {
      display.push(part);
    });
  } else {
    console.log(message);
  }
};
