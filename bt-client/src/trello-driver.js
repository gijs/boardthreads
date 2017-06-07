import Rx from 'rx'
import Trello from 'trello-browser'

const lskey = 'trello-driver-token'

export function makeTrelloDriver (API_KEY, APP_NAME) {
  const trello = new Trello(API_KEY)

  return function trelloDriver (call$) {
    const res$ = new Rx.ReplaySubject(1)

    var token = window.localStorage.getItem(lskey)
    if (token) {
      trello.setToken(token)

      trello.get(`/1/tokens/${token}/dateExpires`)
        .then(expires => res$.onNext({key: 'auth', token}))
        .catch(() => {
          delete trello.token
          window.localStorage.removeItem(lskey)
        })
    }

    call$
      .subscribe(call => {
        if (call.key === 'unauth') {
          window.localStorage.removeItem(lskey)
        } else if (call.key === 'auth') {
          if (trello.token) {
            // we don't perform auth requests if we already have a token
            res$.onNext({key: 'auth', token: trello.token})
            return
          }

          trello.auth({
            name: APP_NAME,
            scope: {read: true, write: true, account: true},
            expiration: '30days'
          })
            .then(body => {
              res$.onNext({key: 'auth', token: trello.token})
              window.localStorage.setItem(lskey, trello.token)
            })
            .catch(err => res$.onError(err))
        } else {
          if (!trello.token) {
            // we don't perform any requests if we do not have a token
            return
          }

          let {method, path, key, data: req} = call
          method = method ? method.toLowerCase() : 'get'
          trello[method](path, req)
            .then(data => res$.onNext({data, key}))
            .catch(err => res$.onError(err))
        }
      })

    return res$
  }
}
