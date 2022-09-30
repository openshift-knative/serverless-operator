// See: https://gist.github.com/simenbrekken-visma/e804c86fd6a23cc59b89913eabbf1d82

const app = window.top;

if (
  app &&
  !app.document.head.querySelector('[data-hide-command-log-request]')
) {
  const style = app.document.createElement('style');
  style.innerHTML =
    '.command-name-request, .command-name-xhr { display: none }';
  style.setAttribute('data-hide-command-log-request', '');

  app.document.head.appendChild(style);
}
