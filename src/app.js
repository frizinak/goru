(function () {
let cbjs = null;
const cbSupport = ClipboardJS.isSupported();
let resultsInit = function () {
    let els = document.getElementsByClassName('img');
    let audios = document.getElementsByTagName('audio');
    for (let i = 0; i < audios.length; i++) {
        audios[i].style.display = 'none';
    }
    let handler = function (e) {
        return function (ev) {
            ev.preventDefault();
            let audio = e.parentElement.parentElement.getElementsByTagName('audio')[0];
            let wasP = audio.paused || audio.ended;
            for (let i = 0; i < audios.length; i++) {
                audios[i].pause();
                audios[i].currentTime = 0;
            }
            if (wasP) {
                audio.volume = 1;
                audio.play();
            }
        };
    };
    for (let i = 0; i < els.length; i++) {
        els[i].onclick = handler(els[i]);
    }

    if (cbjs) {
        cbjs.destroy();
    }
    cbjs = new ClipboardJS('.copy', {
        text: function(e) {
            let cl = 'normal';
            if (e.classList.contains('c-stressed')) {
                cl = 'stressed';
            }
            let el = e.parentElement.parentElement.getElementsByClassName(cl)[0];
            return el.innerText;
        }
    });

    cbjs.on('success', function(e) {
        e.trigger.classList.add('copied');
        setTimeout(function () { e.trigger.classList.remove('copied'); }, 1000);
    });

    cbjs.on('error', function(e) {
        e.trigger.classList.add('error');
        setTimeout(function () { e.trigger.classList.remove('error'); }, 1000);
        e.trigger.innerText = 'failed';
    });

    let copies = document.getElementsByClassName('copy');
    handler = function (ev) { ev.preventDefault(); };
    for (let i = 0; i < copies.length; i++) {
        if (!cbSupport) {
            copies[i].style.display = 'none';
        }
        copies[i].onclick = handler;
    }
};

resultsInit();

let form = document.querySelector('.input form');
let inp = form.getElementsByClassName('val')[0];
let inpWord = function () {
    return inp.value.replace(/^\s+/, '').replace(/\s+$/, '');
};
let absWord = function (w) {
    if (w === '') {
        return '';
    }
    return '/w/' + encodeURIComponent(w);
};
form.onsubmit = function (e) {
    e.preventDefault();
    let w = absWord(inpWord());
    if (w !== '') {
        document.location.pathname = w;
    }
};

let results = document.getElementsByClassName('results')[0];
let lastFetch = inpWord();
let fetchWord = function () {
    let w = inpWord();
    if (w === '' || w === lastFetch) {
        return;
    }

    lastFetch = w;
    fetch(absWord(w), {headers:{'x-requested-with': 'fetch'}}).then(function (res) {
        return res.text();
    }).then(function (t) {
        results.innerHTML = t;
        resultsInit();
        history.replaceState({}, '', absWord(w));
    });
};
setInterval(fetchWord, 500);
})();
