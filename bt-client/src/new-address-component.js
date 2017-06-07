import Rx from 'rx'
import {h} from '@cycle/dom'
import isolate from '@cycle/isolate'

module.exports = (sources, _) => isolate(NewAddressComponent, _)(sources)

function NewAddressComponent ({DOM, trelloInfo$}) {
  let mod$ = Rx.Observable.merge(
    DOM.select('button.start').events('click')
      .do(e => e.preventDefault())
      .map(e => function startMod (data) {
        return data || {}
      }),
    DOM.select('select.board').events('change').merge(
      DOM.select('select.list').events('change')
    )
      .do(e => e.preventDefault())
      .map(e => function selectMod (data) {
        data[e.ownerTarget.name] = e.ownerTarget.value
        return data
      }),
    DOM.select('input.addr').events('input')
      .do(e => e.preventDefault())
      .map(e => function inputMod (data) {
        data[e.ownerTarget.name] = e.ownerTarget.value
        return data
      })
  )
    .startWith(x => x)

  let data$ = mod$.scan((data, mod) => mod(data), null)

  let vtree$ = data$
    .combineLatest(trelloInfo$, (data, trelloinfo) => {
      if (data === null) {
        return h('button.start', 'create a new mailbox')
      } else {
        let {boards = {}, lists = {}} = trelloinfo
        let {board, list, addr} = data
        return h('form.addnew', [
          h('div.group', [
            h('input.addr', {name: 'addr', value: addr}),
            h('input', {disabled: true, value: '@boardthreads.com'})
          ]),
          h('select.board', {name: 'board'},
            [h('option', {key: '-'}, 'choose a board...')].concat(Object.keys(boards)
              .map(b => boards[b])
              .map(b =>
                h('option',
                  {
                    key: b.id,
                    value: b.id,
                    selected: b.id === board
                  },
                  b.name
                )
              )
            )
          ),
          board && h('select.board', {name: 'list'},
            [h('option', 'choose a list')].concat(Object.keys(lists)
              .filter(l => lists[l].idBoard === board)
              .map(l => lists[l])
              .map(l =>
                h('option',
                  {
                    key: l.id,
                    value: l.id,
                    selected: l.id === list
                  },
                  l.name
                )
              )
            )
          ),
          h('button.create', 'Create')
        ])
      }
    })
  .map(el => h('div', [el]))

  let submit$ = DOM.select('form.addnew').events('submit')
    .do(e => e.preventDefault())
    .withLatestFrom(
      data$.filter(data => data !== false),
      (_, data) => data
    )

  return {
    DOM: vtree$,
    submit$
  }
}
