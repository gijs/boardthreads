import Rx from 'rx'
import {h} from '@cycle/dom'
import fwitch from 'fwitch'

import {BASE_DOMAIN, API} from './settings'
import AddressComponent from './address-component'
import NewAddressComponent from './new-address-component'

export default function app ({MAIN, NAV, HTTP, TRELLO}) {
  let menuClick$ = NAV.select('a').events('click')
    .do(e => e.preventDefault())
    .map(e => e.target.id)
    .share()
    .startWith('home')

  let loginResponse$ = HTTP
    .select('login')
    .mergeAll()

  let jwt$ = loginResponse$
    .filter(res => res.status < 300)
    .map(res => res.body.jwt)
    .share()
    .startWith(null)

  let logout$ = Rx.Observable.merge(
    menuClick$.filter(item => item === 'logout'),
    loginResponse$.filter(res => res.status >= 300)
  )
    .map(false)

  let logged$ = Rx.Observable.merge(jwt$, logout$)
    .map(x => !!x)
    .startWith(false)

  let userInfo$ = HTTP
    .select('account')
    .mergeAll()
    .map(res => res.body)
    .share()
    .startWith({addresses: [], lastMessages: []})

  let trelloRequests$ = Rx.Observable.merge(
    MAIN.select('#login').events('click')
      .do(e => e.preventDefault())
      .map({key: 'auth'}),
    logged$
      .filter(logged => logged === true)
      .flatMap([{
        key: 'me',
        path: '/1/member/me',
        data: {'fields': 'username'}
      }, {
        key: 'boards',
        path: '/1/member/me/boards',
        data: {'fields': 'name', 'filter': 'open'}
      }])
      .share(),
    TRELLO
      .filter(t => t.key === 'boards')
      .flatMap(({data: boards}) => boards.map(board => ({
        key: 'lists',
        path: `/1/boards/${board.id}/lists`,
        data: {'fields': 'name,closed,pos,idBoard', 'filter': 'all'}
      })))
      .share(),
    logout$
      .map({key: 'unauth'})
  )

  let trelloInfo$ = TRELLO
    .filter(t => t.key !== 'auth')
    .scan((trelloinfo, {key, data}) => {
      if (key === 'me') {
        trelloinfo.me = data
      } else if (key === 'boards') {
        data.forEach(board => trelloinfo.boards[board.id] = board)
      } else if (key === 'lists') {
        data.forEach(list => trelloinfo.lists[list.id] = list)
      }
      return trelloinfo
    }, {boards: {}, lists: {}, me: {}})
    .debounce(500)
    .startWith({boards: {}, lists: {}, me: {}})

  let addressInfo$ = Rx.Observable.merge(
    HTTP
      .select('addr-info')
      .mergeAll()
      .map(res => res.body),
    HTTP
      .select('set-addr')
      .mergeAll()
      .map(res => res.body)
  )
    .share()

  let addressProps$s$ = Rx.Observable.combineLatest(
    userInfo$,
    trelloInfo$,
    (
      {addresses = []},
      {lists = {}, boards = {}}
    ) => {
      var byAddr = {}
      addresses.forEach(addr => {
        byAddr[addr.inboundaddr] = addressInfo$
          .filter(info => info.inboundaddr === addr.inboundaddr)
          .startWith(addr)
          .map(addr => {
            let list = lists[addr.listId]
            let board = boards[list && list.idBoard]
            return {addr, list, board}
          })
      })
      return byAddr
    }
  )

  let addnew = NewAddressComponent({
    DOM: MAIN,
    trelloInfo$
  })

  let addresses$ = addressProps$s$
    .map(props$s => {
      return Object.keys(props$s).map(inboundaddr => {
        let a = AddressComponent({
          DOM: MAIN,
          trelloInfo$,
          props$: props$s[inboundaddr]
        })
        a.key = inboundaddr
        return a
      })
    })
    .share()
    .startWith([])

  let main$ = Rx.Observable.combineLatest(
    logged$,
    userInfo$,
    menuClick$,
    addresses$,
    trelloInfo$,
    (logged, {lastMessages}, clicked, addresses, {me}) => {
      /* the actual information being shown */
      var body
      switch (clicked) {
        case 'home':
          var c
          if (lastMessages.length === 0) {
            c = h('p', `You haven't sent or received any message yet.`)
          } else {
            c = h('ul.activity-log', lastMessages.map(m =>
              h('li', {key: m.id}, [
                h('article', [
                  h('header', [
                    h('h1', [
                      m.from
                        ? '↓' // arrow down
                        : '↑', // arrow up
                      h('a', {href: `https://trello.com/c/${m.cardShortLink}`, target: '_blank'}, m.subject.trim() || '(no subject)')
                    ])
                  ]),
                  h('div', [
                    h('p', (m.from
                      ? ['received from ', h('b', m.from), ' at ', h('b', m.address)]
                      : ['sent from account ', h('b', m.address)]
                    ).concat([' with message ID ', h('b', m.id)])),
                    h('p', [
                      'comment ',
                      h('a', {href: `https://trello.com/c/${m.cardShortLink}#comment-${m.commentId}`, target: '_blank'}, m.commentId),
                      ' at card ',
                      h('a', {href: ``, target: '_blank'}, m.cardShortLink),
                      '.'
                    ])
                  ])
                ])
              ])
            ))
          }
          body = [
            h('h1', 'Recent activity from your addresses'),
            c
          ]
          break
        case 'addresses':
          body = [
            h('ul', addresses.map(a => h('li', {key: a.key}, a.DOM))),
            addnew.DOM
          ]
          break
        default:
          body = h('div')
      }
      /* ~ */

      /* proceed to render */
      if (!logged) {
        return h('article', [
          h('button', {id: 'login'}, 'click here to login')
        ])
      } else {
        return h('section', [
          h('header', [
            h('h1', clicked.toUpperCase())
          ])
        ].concat(body))
      }
      /* ~ */
    }
  )

  let nav$ = Rx.Observable.combineLatest(
    logged$,
    menuClick$,
    trelloInfo$,
    (logged, clicked, {me = {username: ''}}) => {
      if (!logged) {
        return h('ul')
      } else {
        return h('ul', [
          h('li', [
            h('a', me.username)
          ])
        ].concat(menuItems.map(item =>
          h('li',
            h('a',
              {className: clicked === item ? 'selected' : '',
               id: item,
               href: '#/'},
              item
            )
          )
        )))
      }
    }
  )

  let request$ = Rx.Observable.merge(
    // after getting a token from trello, login on boardthreads
    TRELLO
      .filter(t => t.key === 'auth')
      .map(({token}) => ({
        category: 'login',
        url: API + '/api/session',
        method: 'POST',
        send: {trello_token: token}
      })),
    // get boardthreads account info
    Rx.Observable.merge(
      logged$.filter(logged => logged === true),
      HTTP.select('delete-addr'),
      HTTP.select('downgrade-addr'),
      HTTP.select('new-addr')
    )
      .delay(1000)
      .share()
      .map(() => ({category: 'account', url: API + '/api/account'})),
    // create a new form (from `addnew`)
    addnew.submit$
      .map(({addr, list}) => ({
        category: 'new-addr',
        url: API + '/api/addresses/' + addr,
        method: 'PUT',
        send: {listId: list}
      })),
    // addresses being opened require more info
    addresses$
      .flatMap((addresses = []) => addresses.map(a => a.moreInfo$))
      .mergeAll()
      .map(address => ({
        category: 'addr-info',
        url: API + '/api/addresses/' + address
      })),
    // changing an address
    addresses$
      .flatMap((addresses = []) => addresses.map(a => a.change$))
      .mergeAll()
      .map(addrInfo => ({
        category: 'set-addr',
        url: API + '/api/addresses/' + addrInfo.inboundaddr.split('@')[0],
        method: 'PUT',
        send: addrInfo
      })),
    addresses$
      .flatMap((addresses = []) => addresses.map(a => a.changeSettings$))
      .mergeAll()
      .map(params => ({
        category: 'change-settings',
        url: API + '/api/addresses/' + params.inboundaddr.split('@')[0] + '/settings',
        method: 'PUT',
        send: params
      })),
    // upgrading addresses
    addresses$
      .flatMap((addresses = []) => addresses.map(a => a.upgrade$))
      .mergeAll()
      .map(address => ({
        category: 'upgrade-addr',
        url: API + '/billing/' + address + '/paypal'
      })),
    // downgrading addresses
    addresses$
      .flatMap((addresses = []) => addresses.map(a => a.downgrade$))
      .mergeAll()
      .map(address => ({
        category: 'downgrade-addr',
        url: API + '/billing/' + address + '/paypal',
        method: 'DELETE'
      })),
    // deleting addresses
    addresses$
      .flatMap((addresses = []) => addresses.map(a => a.delete$))
      .mergeAll()
      .map(address => ({
        category: 'delete-addr',
        url: API + '/api/addresses/' + address,
        method: 'DELETE'
      })),
    // checking dns for domain
    addresses$
      .flatMap((addresses = []) => addresses.map(a => a.checkDNS$))
      .mergeAll()
      .map(domain => ({
        category: 'check-dns',
        url: API + '/api/check-dns/' + domain,
        method: 'POST'
      }))
  )
    .share()

  let notify$ = Rx.Observable.merge(
    HTTP
      .select('new-addr')
      .mergeAll()
      .catch([`<p>You don't have permission to add members to this board.</p><p> We need these permissions to add <a style="color: white" href='https://trello.com/boardthreads'>our bot</a> that will create the cards for the received email messages.</p>`, {addnCls: 'humane-flatty-error'}]),
    HTTP
      .select('set-addr')
      .mergeAll()
      .filter(res =>
        res.request.send.outboundaddr &&
        res.request.send.outboundaddr.split('@')[1] !== BASE_DOMAIN &&
        res.request.send.outboundaddr !== res.body.outboundaddr
      )
      .map([`<p>This domain is controlled by another user, you can't use it on your own addresses.</p> If you don't know what this means or don't agree with it, please contact us.`, {addnCls: 'humane-flatty-error'}]),
    HTTP
      .select('upgrade-addr')
      .flatMap(r$ => r$
        .map(null)
        .catch(e => Rx.Observable.just([`<p>${fwitch(e.status, {
          403: "A strange error has happened. We couldn't verify the ownership of the address you're trying to upgrade.",
          default: 'Something wrong happening while we were trying to redirect you to Paypal. Please contact us.'
        })}</p>`, {addnCls: 'humane-flatty-error'}]))
      ),
    HTTP
      .select('change-settings')
      .mergeAll()
      .map(['Preferences saved!', 'success']),
    HTTP
      .select('check-dns')
      .mergeAll()
      .map('DNS is being checked, please wait a few seconds and refresh the page.')
  )

  return {
    MAIN: main$,
    NAV: nav$,
    HTTP: request$
      .withLatestFrom(jwt$, (req, jwt) => {
        if (jwt) req.headers = {'Authorization': `Bearer ${jwt}`}
        return req
      }),
    TRELLO: trelloRequests$,
    NOTIFICATION: notify$,
    REDIRECT: HTTP
      .select('upgrade-addr')
      .flatMap(r$ => r$
        .map(res => res.text)
        .catch(Rx.Observable.empty())
      )
  }
}

const menuItems = ['home', 'addresses', 'logout']
