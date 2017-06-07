import requests

MAILGUN_API_KEY = 'key-fa9064ccb0c0f07d86b89d27ea658a89'
MAILGUN_URL = 'https://api.mailgun.net/v3'

addresses = [
  'cardsync@boardthreads.com',
  'sillicon13@boardthreads.com',
  'cornerboothcafe@boardthreads.com',
  'focuscentric@boardthreads.com',
  'et@boardthreads.com',
  'etools@boardthreads.com',
  'welcome@boardthreads.com',
  'bjstrevy@boardthreads.com',
  'test@boardthreads.com',
  'hiisi@boardthreads.com',
  'autumnsalad@boardthreads.com',
  'help@boardthreads.com',
  'wildpoetry@boardthreads.com',
  'weatheredtest@boardthreads.com',
  'tttsupport@boardthreads.com',
  'catchall@boardthreads.com',
  'weathereddust@boardthreads.com',
  'summerscene@boardthreads.com',
  'afolabi989@boardthreads.com',
  'namelessdust@boardthreads.com',
  'hospedariaabp@boardthreads.com',
  'dynamicsonline@boardthreads.com',
  'wildbutterfly@boardthreads.com',
  'drylimit@boardthreads.com',
  'prhunters@boardthreads.com',
  'supportdjolaq@boardthreads.com',
  'nimbis@boardthreads.com',
  'shawntaylor@boardthreads.com',
  'formspree@boardthreads.com',
  'stillfirefly@boardthreads.com',
  'sol@boardthreads.com',
  'dooverygang@boardthreads.com',
  'tareq@boardthreads.com',
  'segment@boardthreads.com',
  'luvdasun-support@boardthreads.com',
  'rwsnw@boardthreads.com',
  'august@boardthreads.com',
  'team@formspree.io',
  'fiatjaf@boardthreads.com',
  'remus-radvan@boardthreads.com',
  'support@budstv.org',
  'budstv@boardthreads.com',
  'fragrantsun@boardthreads.com',
  'prodsystems@boardthreads.com',
  'nimbis-awesim@boardthreads.com',
  'awesim-support@boardthreads.com',
  'support@boardthreads.com',
  'support@marketingpartner.be',
  'baldwin-helpdesk@boardthreads.com',
  'doublemap@boardthreads.com',
  'support@doublemap.com',
  'fragrantlimit@boardthreads.com',
  'flatfashion@boardthreads.com',
  'whitesunset@boardthreads.com',
  'geismar@boardthreads.com',
  'stefan@boardthreads.com',
  'demandlab@boardthreads.com',
  'eduguide@boardthreads.com',
  'mboengineering@boardthreads.com',
  'drythunder@boardthreads.com',
  'supers@boardthreads.com',
  'sfh@boardthreads.com',
  'super@boardthreads.com',
  'super@lactuel.ca',
  'superpixel@boardthreads.com',
  'sems@boardthreads.com',
  'events@soarflylogistics.com',
  'vtc@soarflylogistics.com',
  'h64@boardthreads.com',
  'support@superpixel.co',
  'a-testing-address@boardthreads.com',
  'alquimia@boardthreads.com',
  'salesatfftec@boardthreads.com',
  'websitesfortrello@boardthreads.com',
  'support@formspree.io',
  'fiatjaf@alhur.es',
  'abp@alhur.es',
  'alquimia@alhur.es',
  'fiatjaf@alhur.e',
  'www@alhur.es',
  'fiatjaf@inputs.space',
  'fiatjaf@xarope.com',
  'fiatjaf@banana.com',
  'fiatjaf@wekwerewr.com',
  'fiatjaf@wft.space',
  'alquimia@wft.space',
  'bot@boardthreads.com',
  'sales@fftec.com',
  'aaronpatfftec@boardthreads.com',
  'aaronpatfftec@fftec.com',
  'peach-bugs@boardthreads.com',
  'mile18-service@boardthreads.com',
  'info@epos-schweiz.ch',
  'lukas.grueter@boardthreads.com',
  'lukas.grueter@epos-schweiz.ch',
  'itsupport-maximefiset@boardthreads.com',
  'spiffmedia@boardthreads.com',
  'undefined@boardthreads.com',
  'sheets@boardthreads.com',
  'demo@boardthreads.com',
  'spiri@boardthreads.com',
  'wepack@boardthreads.com',
  'purple@boardthreads.com',
  'purple-api@boardthreads.com',
  'celil@boardthreads.com',
  'bbhopline@boardthreads.com',
  'bbsurrender@boardthreads.com',
  'bbfoster@boardthreads.com',
  'lesrwood@boardthreads.com',
  'support_sol@boardthreads.com',
  'support_sol@gticanada.com',
  'fishbloom@boardthreads.com',
  'cait@webmarketers.ca',
  'support@webmarketers.ca',
  'support@webmarketersdev.ca',
  'mnlsupply@boardthreads.com',
  'mnlsupply@webmarketers.support',
  'webmarketers@boardthreads.com',
  'help@support.webmarketers.ca',
  'cardsync-v2-test@boardthreads.com',
  'testnts@boardthreads.com',
  'contact@support.webmarketers.ca',
  'example@boardthreads.com',
  'suzanne@rawnet.com',
  'mjesman@boardthreads.com',
  'panorama_usa@boardthreads.com',
  'requestlog@panorama-antennas.com',
  'sitesol-help@boardthreads.com',
  'help@sitesol.ca',
  'henchling-support@boardthreads.com',
  'sentinel@boardthreads.com',
  'helpdesk@providenceeng.com',
  'low@boardthreads.com',
  'medium@boardthreads.com',
  'high@boardthreads.com',
  'morph123@boardthreads.com',
  'colefabrics-servicedesk@boardthreads.com',
  'cardsync-green@boardthreads.com',
  'cardsync-red@boardthreads.com',
  'cardsync-v2-test-green@boardthreads.com',
  't&r@boardthreads.com',
  'help@colefabrics.com',
  'devops@boardthreads.com',
  'dudas@boardthreads.com',
  'thewebsiteshop@boardthreads.com',
  'support@thewebsiteshop.ie',
  'support_tm@boardthreads.com',
  'support_tm@gticanada.com',
  'grandview-maintenance@boardthreads.com',
  'herosupport@boardthreads.com',
  'yreceiptslab@boardthreads.com',
  'helpmejcc@boardthreads.com',
  'jalapeno@boardthreads.com',
  'erptechintegration@boardthreads.com',
  'erptechintegration@tylertech.com',
  'iman@boardthreads.com',
  'iman@crowdskills.com',
  'info@boardthreads.com',
  'lhs@boardthreads.com',
  'clinton@boardthreads.com'
]

for addr in addresses:
    r = requests.post(
        MAILGUN_URL + '/boardthreads.com/messages',
        auth=('api', MAILGUN_API_KEY),
        data={
            'from': 'BoardThreads Announcement <help@boardthreads.com>',
            'to': addr,
            'subject': 'BoardThreads will shut down.',
            'text': '''
Hello, dear BoardThreads user.

It's been a nice journey, the last 2 years, or more, I don't know exactly.
But we're going to shut down. We only had a handful of paying users for this
whole time, and never actually got the service to the level of features and
functionalities we wished.

Our last standing agreement ends at August 3, so we're gonna close the doors
at August 4. If you have an active billing subscription we're going to cancel
it automatically at July 3 (your account will continue to work for the due
period of your last billing month).

We're also going to open-source the code for the entire platform at
https://github.com/fiatjaf/boardthreads. So if you really want to keep using
BoardThreads you'll only need a cheap server, a Mailgun account and a Neo4j
database. Reach me if you want help setting up a server.

We are glad to hear from you if you have any doubts or whatever else you want
to say.

Giovanni T. Parra
            ''',
            'html': '''
<p>Hello, dear BoardThreads user.</p>

<p>It's been a nice journey, the last 2 years, or more, I don't know exactly.
But we're going to shut down. We only had a handful of paying users for this
whole time, and never actually got the service to the level of features and
functionalities we wished.</p>

<p>Our last standing agreement ends at August 3, so we're gonna close the doors
at August 4. If you have an active billing subscription we're going to cancel
it automatically at July 3 (your account will continue to work for the due
period of your last billing month).</p>

<p>We're also going to open-source the code for the entire platform at
https://github.com/fiatjaf/boardthreads. So if you really want to keep using
BoardThreads you'll only need a cheap server, a Mailgun account and a Neo4j
database. Reach me if you want help setting up a server.</p>

<p>We are glad to hear from you if you have any doubts or whatever else you
want to say.</p>

<p>Giovanni T. Parra</p>
            '''
        }
    )
    print(r.text)
