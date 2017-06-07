import {h} from '@cycle/dom'

function ValueHook (value) {
  this.value = value
}
ValueHook.prototype.hook = function (node) {
  node.value = this.value
}

const vtree = {}

vtree.editList = (lists, boards, list, board, editingList, selectedBoard) => editingList
  ? h('div.change-list', [
    h('form', [
      h('select.board', Object.keys(boards).map(b =>
        h('option', {value: boards[b].id, selected: boards[b].id === board}, boards[b].name)
      )),
      ' ',
      board && h('select.list', Object.keys(lists)
        .filter(l => lists[l].idBoard === (selectedBoard || board))
        .map(l =>
          h('option', {value: lists[l].id, selected: lists[l].id === list}, lists[l].name)
        )),
      ' ',
      h('button', 'Change')
    ])
  ])
  : null

vtree.setOutbound = addr =>
  h('form.outboundaddr', [
    h('h1', 'Send mail from the following address:'),
    h('input', {
      value: new ValueHook(addr.outboundaddr || addr.inboundaddr)
    }),
    ' ',
    h('button', 'Use')
  ])

vtree.markReceiveMail = (addr, showMxRecords) =>
  addr.domain && addr.domain.dns
    ? h('label.show-mx.hint--bottom', {dataset: {hint: `As long as you have the MX records properly set, this box will be checked and the emails you send will have the Reply-To header set to ${addr.outboundaddr}.`}}, [
      'Also receive mail at this address ',
      h('input', {type: 'checkbox', checked: addr.domain && addr.domain.dns && showMxRecords})
    ])
    : null

vtree.settings = addr =>
  h('form.settings', [
    h('h1', 'Preferences'),
    h('label.senderName', {className: 'hint--top', dataset: {hint: 'The name your customers will see preceding your email address.'}}, [
      'From name',
      h('input', {value: addr.settings.senderName})
    ]),
    h('label.replyTo', {className: 'hint--top', dataset: {hint: `Replies to the messages you sent will be sent to this address. If you are not using BoardThreads to receive mail with your custom domain and you are redirecting mail received at another address to BoardThreads, you should write it here. Otherwise don't change this field.`}}, [
      'Reply-To address',
      h('input', {value: addr.settings.replyTo})
    ]),
    h('label.addReplier', {className: 'hint--top', dataset: {hint: 'Check this if you want BoardThreads to add the user who replied to a card on Trello as a member of the card.'}}, [
      'Add replier as a member of the card',
      h('input', {type: 'checkbox', checked: addr.settings.addReplier})
    ]),
    h('label.moveToTop', {className: 'hint--top', dataset: {hint: 'Enabling this option will ensure that a card is moved back to the top of the list whenever a new email message comes in.'}}, [
      'Move cards to the top of the list',
      h('input', {type: 'checkbox', checked: addr.settings.moveToTop})
    ]),
    h('label.messageInDesc', {className: 'hint--top', dataset: {hint: 'This option will make the first message of the card to be written in the card\'s description, besides being posted as a comment.'}}, [
      'First message of thread in the card description',
      h('input', {type: 'checkbox', checked: addr.settings.messageInDesc})
    ]),
    h('label.signatureTemplate', {className: 'hint--top', dataset: {hint: 'A markdown template to be appended to the bottom of each reply. You can use the variables {NAME} and {USERNAME} to identify the agent who is answering. These values correspond to the values of each personal Trello account.'}}, [
      'Signature template',
      h('textarea', {placeholder: `---

Cheers,

[{NAME}](https://trello.com/{USERNAME})
Company Name`, value: addr.settings.signatureTemplate})
    ]),
    ' ',
    h('button', 'Save')
  ])

vtree.sendingDNS = addr =>
  addr.domain && addr.domain.dns
    ? h('div', [
      h('h1', 'Add the following DNS records to start sending'),
      h('div.dns', [
        vtree.dnsRow(addr.domain.dns.include),
        vtree.dnsRow(addr.domain.dns.domain_key)
      ]),
      h('small', addr.domain.dns.include.valid && addr.domain.dns.domain_key.valid
        ? `As long as these records are valid, messages sent from this list will be automatically sent from ${addr.outboundaddr}.`
        : `While these records are not confirmed, your emails will be sent from ${addr.inboundaddr}.`
      )
    ])
    : null

vtree.receivingDNS = (addr, showMxRecords) =>
  addr.domain && addr.domain.dns && showMxRecords
    ? h('div', [
      h('h1', 'and the following DNS records to start receiving'),
      h('div.dns',
        addr.domain.dns.receive
          .map(record => (record.name = addr.domain.name) && record)
          .map(record => vtree.dnsRow(record))
      ),
      h('small', addr.domain.dns.receive.filter(r => r.valid).length === 2
        ? `As long as these records are valid, messages received at ${addr.outboundaddr} will be redirected to this list, and messages sent from this list will have the Reply-To header set to ${addr.outboundaddr}. Messages sent to ${addr.inboundaddr} will still work.`
        : `While these records are not found, the emails sent from your address will go will have the Reply-To header set to ${addr.inboundaddr}.`)
    ])
    : null

vtree.dnsRow = record => h('div', [
  h('span', record.type),
  h('span', record.name),
  h('span', record.value),
  record.priority ? h('span', record.priority) : null,
  h(
    'span.hint--top',
    {dataset: {hint: record.valid
      ? 'Confirmed!'
      : "DNS record wasn't confirmed yet"
    }},
    record.valid
      ? h('span.typcn.typcn-tick-outline')
      : h('span.typcn.typcn-times-outline')
  )
])

vtree.checkDNS = addr =>
  addr.domain
    ? h('div.check-dns', [
      h('button', 'Check DNS Records for ' + addr.domain.name)
    ])
    : null

module.exports = vtree
