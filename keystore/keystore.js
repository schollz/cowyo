function save(key, value) {
  c = new WebSocket('wss://cowyo.com/ws');
  return new Promise(function (resolve, reject) {
    try {
      c.onopen = function (_) {
        c.send(JSON.stringify({
          TextData: JSON.stringify(value),
          Title: `${key}`,
          UpdateServer: true,
          UpdateClient: false,
        }));
        return resolve(true);
      }
    } catch(e) {
      return reject(e);
    }
  });
}

// save('hello2', 'world');




function get(key) {
 c = new WebSocket('wss://cowyo.com/ws');
 return new Promise(function (resolve, reject) {
   try {
     c.onmessage = function(evt) {
       return resolve(JSON.parse(JSON.parse(evt.data).TextData));
     }
     c.onopen = function (_) {
       c.send(JSON.stringify({
         Title: `${key}`,
         UpdateClient: true,
       }));
     };
   } catch(e) {
     return reject(e);
   }
 });
}
