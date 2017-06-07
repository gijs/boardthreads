import Rx from 'rx'
import {h} from '@cycle/dom'
import isolate from '@cycle/isolate'
import fwitch from 'fwitch'
import cx from 'class-set'

import vtree from './vtree'

module.exports = (sources, _) => isolate(AddressComponent, _)(sources)

function AddressComponent ({DOM, trelloInfo$, props$}) {
  let opened$ = DOM.select('header > h1 > a:first-child').events('click')
    .do(e => e.preventDefault())
    .scan((opened) => !opened, false)
    .share()

  let isSomeMxRecordSet$ = props$
    .map(({addr}) =>
      addr.domain &&
      addr.domain.dns &&
      addr.domain.dns.receive &&
      addr.domain.dns.receive.filter(rec => rec.valid).length > 1)

  let vtree$ = Rx.Observable.combineLatest(
    props$,
    trelloInfo$,
    opened$
      .startWith(false),
    DOM.select('.list').events('click')
      .do(e => e.preventDefault())
      .scan(editingList => !editingList, false)
      .startWith(false),
    DOM.select('form.change-list select.board').events('change')
      .map(e => e.ownerTarget.value)
      .startWith(null),
    DOM.select('.show-mx input[type="checkbox"]').events('change')
      .map(e => e.ownerTarget.checked)
      .merge(isSomeMxRecordSet$),
    (
      {addr, list, board},
      {boards = {}, lists = {}},
      opened,
      editingList,
      selectedBoard,
      showMxRecords
    ) => addr && list && board
      ? h('article', {className: cx({opened}, addr.status)}, [
        h('header', [
          h('h1', [
            h('a', {
              href: '#',
              className: cx({'hint--right': !opened}),
              dataset: opened ? {hint: ''} : {hint: fwitch(addr.status, {
                DISABLED: 'The trial period for this address has expired. Click to set up payments.',
                VALID: 'This address has payments enabled. Click to configure.',
                TRIAL: 'This address is in the trial period. Click to configure or set payments.'
              })}
            }, addr.inboundaddr),
            ` ⇝ `,
            h('a', {
              target: '_blank',
              href: `https://trello.com/b/${board.id}`
            }, board.name),
            ` / `,
            h('span.list', list.name)
          ]),
          !opened ? null : h('aside', [
            h('ul', [
              h('li', [
                h('a.delete', 'delete address')
              ]),
              h('li', [
                fwitch(addr.status, {
                  VALID: h('a.downgrade', 'Deactivate address'),
                  TRIAL: h('a.upgrade', 'Upgrade address'),
                  DISABLED: h('a.upgrade', 'Enable address')
                })
              ])
            ])
          ])
        ]),
        opened
          ? h('div', [
            vtree.editList(lists, boards, list, board, editingList, selectedBoard),
            vtree.setOutbound(addr),
            vtree.markReceiveMail(addr, showMxRecords),
            vtree.sendingDNS(addr, showMxRecords),
            vtree.receivingDNS(addr, showMxRecords),
            vtree.checkDNS(addr),
            vtree.settings(addr)
          ])
          : null
      ])
      : h('div')
    )

  let moreInfo$ = opened$
    .filter(opened => opened === true)
    .withLatestFrom(props$, (_, {addr}) => addr.inboundaddr.split('@')[0])

  let delete$ = DOM.select('.delete').events('click')
    .do(e => e.preventDefault())
    .withLatestFrom(props$, (_, {addr}) => addr.inboundaddr.split('@')[0])

  let downgrade$ = DOM.select('.downgrade').events('click')
    .do(e => e.preventDefault())
    .withLatestFrom(props$, (_, {addr}) => addr.inboundaddr.split('@')[0])

  let upgrade$ = DOM.select('.upgrade').events('click')
    .do(e => e.preventDefault())
    .withLatestFrom(props$, (_, {addr}) => addr.inboundaddr.split('@')[0])

  let changeList$ = DOM.select('.change-list form').events('submit')
    .do(e => e.preventDefault())
    .map(e => ({
      key: 'listId',
      value: e.ownerTarget.querySelector('list').value
    }))

  let changeOutboundAddr$ = DOM.select('form.outboundaddr').events('submit')
    .do(e => e.preventDefault())
    .map(e => ({
      key: 'outboundaddr',
      value: e.ownerTarget.querySelector('input').value
    }))

  let checkDNS$ = DOM.select('.check-dns button').events('click')
    .do(e => e.preventDefault())
    .withLatestFrom(props$, (_, {addr}) => addr.domain.name)

  let changeSettings$ = DOM.select('form.settings').events('submit')
    .do(e => e.preventDefault())
    .map(e => ({
      senderName: e.ownerTarget.querySelector(`.senderName input`).value,
      replyTo: e.ownerTarget.querySelector(`.replyTo input`).value,
      addReplier: e.ownerTarget.querySelector(`.addReplier input`).checked,
      messageInDesc: e.ownerTarget.querySelector('.messageInDesc input').checked,
      moveToTop: e.ownerTarget.querySelector('.moveToTop input').checked,
      signatureTemplate: e.ownerTarget.querySelector('.signatureTemplate textarea').value
    }))
    .withLatestFrom(props$, (params, props) => (params.inboundaddr = props.addr.inboundaddr) && params)

  let change$ = Rx.Observable.merge(changeList$, changeOutboundAddr$)
    .withLatestFrom(props$, (change, props) => {
      let newAddressInfo = props.addr
      newAddressInfo[change.key] = change.value
      return newAddressInfo
    })

  return {
    DOM: vtree$,
    moreInfo$,
    downgrade$,
    upgrade$,
    delete$,
    checkDNS$,
    change$,
    changeSettings$
  }
}
