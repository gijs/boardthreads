export const BASE_DOMAIN = 'boardthreads.com'

export const API = window.location.port
  ? window.location.origin.replace('000', '001')
  : 'https://api.boardthreads.com'
