import humane from 'humane-js'

export function makeHumaneDriver (settings = {}) {
  const logger = humane.create(settings)

  // show notifications based on hash
  if (window.location.hash) {
    let [type, message] = window.location.hash.slice(1).split('=')
    if (message) {
      window.location.hash = ''
      logger.log(message, {addnCls: 'humane-flatty-' + type})
    }
  }

  return function humaneDriver (message$) {
    message$
      .filter(m => m)
      .subscribe(message => {
        if (typeof message === 'string') {
          logger.log(message)
        } else if (Array.isArray(message)) {
          if (typeof message[1] === 'object') {
            logger.log(message[0], message[1])
          } else {
            logger.log(message[0], {addnCls: 'humane-flatty-' + message[2]})
          }
        }
      })
  }
}
