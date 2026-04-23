(()=>{function k(q){if(typeof document>"u"){q();return}if(document.readyState==="loading"){document.addEventListener("DOMContentLoaded",q,{once:!0});return}q()}function M(q,z=document){return z.querySelectorAll(`[data-${q}]`)}var y='a[href], button, input, select, textarea, [tabindex]:not([tabindex="-1"])';function D(q){return Array.from(q.querySelectorAll(y)).filter((J)=>!J.hasAttribute("disabled")&&J.getAttribute("aria-hidden")!=="true")}function U(){return typeof document<"u"&&typeof window<"u"}var S="__UI8KIT_ARIA_AUTO_INIT__";function A(){return globalThis[S]!==!1}var P="ui8kit",$={patterns:new Map,scopes:new Map},Y=null,_={ready:k,byAttr:M};function C(q){return q!=null?q:U()?document:{}}function E(){let q=globalThis[P];if(q)return q;let z={core:_,ready:k,byAttr:M,register:w,init:O,initPattern:K};if(U())globalThis[P]=z;return z}function f(){if(!Y)Y=E();return Y}function w(q){if($.patterns.has(q.name))return;if($.patterns.set(q.name,q),f()[q.name]=q,A()&&U())k(()=>{K(q.name)})}function K(q,z){var J;let Q=$.patterns.get(q);if(!Q)return;if(!U()&&!z)return;let W=C(z),Z=(J=$.scopes.get(q))!=null?J:new WeakMap;if($.scopes.set(q,Z),Z.has(W))return;let j=Q.init(W);Z.set(W,typeof j==="function"?j:R)}function O(q){for(let z of $.patterns.values())K(z.name,q)}function R(){}var G="data-state",L="open",b="closed",H='[data-ui8kit="dialog"], [data-ui8kit="sheet"], [data-ui8kit="alertdialog"]',B='[data-ui8kit="dialog"][data-state="open"], [data-ui8kit="sheet"][data-state="open"], [data-ui8kit="alertdialog"][data-state="open"]',V=new WeakMap;function T(q){var z,J;if(!q)return null;if(typeof q==="string")return document.getElementById(q);let Q=q;if(typeof Q.matches==="function"&&Q.matches(H))return Q;let W=(z=Q.getAttribute)==null?void 0:z.call(Q,"data-ui8kit-dialog-target");if(W)return document.getElementById(W);return(J=Q.closest)==null?void 0:J.call(Q,H)}function N(q,z){if(!q)return;let J=q.querySelector("[data-ui8kit-dialog-overlay]");if(z){q.setAttribute(G,L),q.removeAttribute("hidden"),J==null||J.removeAttribute("hidden");let Q=D(q)[0];Q==null||Q.focus(),q.dataset.trapped="1"}else{q.setAttribute(G,b),q.setAttribute("hidden","hidden"),J==null||J.setAttribute("hidden","hidden");let Q=V.get(q);if(Q&&document.contains(Q))Q.focus();V.delete(q),delete q.dataset.trapped}}function x(q){let z=T(q);if(!z)return;let J=q&&q instanceof HTMLElement&&q!==z?q:document.activeElement;if(J)V.set(z,J);N(z,!0)}function X(q){N(T(q),!1)}function I(q){if(q.key==="Escape"){let j=document.querySelector(B);if(j)X(j),q.preventDefault();return}if(q.key!=="Tab")return;let z=document.querySelector(B);if(!z||!z.contains(q.target))return;let J=D(z);if(J.length===0){q.preventDefault();return}let Q=J[0],W=J[J.length-1];if(!Q||!W){q.preventDefault();return}let Z=document.activeElement;if(q.shiftKey){if(Z===Q||Z===z)W.focus(),q.preventDefault()}else if(Z===W||Z===z)Q.focus(),q.preventDefault()}function F(q){let z=q.target;if(!z)return;let J=z.closest("[data-ui8kit-dialog-open]");if(J){x(J),q.preventDefault();return}if(z.closest("[data-ui8kit-dialog-close]")){X(z.closest(H)),q.preventDefault();return}if(z.closest("[data-ui8kit-dialog-overlay]"))X(z.closest(H))}var h={name:"dialog",init(q=document){let J=q.querySelectorAll(H);for(let Q of J)N(Q,Q.getAttribute(G)===L);return document.addEventListener("click",F),document.addEventListener("keydown",I),()=>{document.removeEventListener("click",F),document.removeEventListener("keydown",I)}},open:x,close:X};w(h);if(typeof window<"u"&&window.__UI8KIT_ARIA_AUTO_INIT__!==!1)f().init();})();
;
(function () {
  var namespace = window.ui8kit || {};
  window.ui8kit = namespace;
  if (namespace.languageSwitch) {
    return;
  }

  function ready(fn) {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", fn);
      return;
    }
    fn();
  }

  function parseResponse(html) {
    var parser = new DOMParser();
    return parser.parseFromString(html, "text/html");
  }

  function replaceMainContent(button, html) {
    var targetSelector = button.getAttribute("data-spa-target") || "main";
    var parsed = parseResponse(html);

    var currentTarget = document.querySelector(targetSelector);
    var nextTarget = parsed.querySelector(targetSelector);
    if (currentTarget && nextTarget) {
      currentTarget.innerHTML = nextTarget.innerHTML;
    }

    var parsedTitle = parsed.querySelector("title");
    if (parsedTitle && parsedTitle.textContent) {
      document.title = parsedTitle.textContent;
    }

    var nextButton = button.id ? parsed.getElementById(button.id) : null;
    if (nextButton) {
      if (nextButton.getAttribute("href")) {
        button.setAttribute("href", nextButton.getAttribute("href"));
      }
      if (nextButton.dataset.currentLocale) {
        button.dataset.currentLocale = nextButton.dataset.currentLocale;
      }
      if (nextButton.dataset.nextLocale) {
        button.dataset.nextLocale = nextButton.dataset.nextLocale;
      }
      button.textContent = nextButton.textContent;
    }

    var locale = parsed.documentElement && parsed.documentElement.getAttribute("lang");
    if (locale) {
      document.documentElement.setAttribute("lang", locale);
    }
  }

  function bindLanguageSwitch(button) {
    button.addEventListener("click", function (event) {
      event.preventDefault();
      var href = button.getAttribute("href");
      if (!href) {
        return;
      }

      fetch(href, {
        credentials: "same-origin",
        headers: {
          "X-Locale-Switch": "1",
        },
      })
        .then(function (response) {
          if (!response.ok) {
            throw new Error("locale switch request failed");
          }
          return response.text();
        })
        .then(function (html) {
          replaceMainContent(button, html);
          history.pushState({}, "", href);
        })
        .catch(function () {
          window.location.href = href;
        });
    });
  }

  ready(function () {
    var toggles = document.querySelectorAll("[data-ui8kit-spa-lang]");
    for (var i = 0; i < toggles.length; i += 1) {
      bindLanguageSwitch(toggles[i]);
    }
  });

  namespace.languageSwitch = { init: function () {} };
})();
