package com.xiaofei.liveagent

import android.graphics.Color
import android.os.Bundle
import android.util.TypedValue
import android.view.View
import android.view.ViewGroup
import android.webkit.JavascriptInterface
import android.webkit.WebView
import android.widget.FrameLayout
import androidx.activity.enableEdgeToEdge
import androidx.annotation.Keep
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.webkit.WebViewCompat
import androidx.webkit.WebViewFeature
import kotlin.math.abs
import kotlin.math.roundToInt

class MainActivity : TauriActivity() {
  private var appWebView: WebView? = null
  private var safeAreaHost: SafeAreaHost? = null

  override fun onCreate(savedInstanceState: Bundle?) {
    enableEdgeToEdge()
    super.onCreate(savedInstanceState)
  }

  override fun onWebViewCreate(webView: WebView) {
    super.onWebViewCreate(webView)
    appWebView = webView
    webView.addJavascriptInterface(SafeAreaBridge(), BRIDGE_NAME)
    if (WebViewFeature.isFeatureSupported(WebViewFeature.DOCUMENT_START_SCRIPT)) {
      WebViewCompat.addDocumentStartJavaScript(webView, THEME_SCRIPT, setOf("*"))
    }
    if (webView.isAttachedToWindow) {
      installSafeAreaHost(webView)
    } else {
      webView.addOnAttachStateChangeListener(
        object : View.OnAttachStateChangeListener {
          override fun onViewAttachedToWindow(v: View) {
            v.removeOnAttachStateChangeListener(this)
            installSafeAreaHost(webView)
          }

          override fun onViewDetachedFromWindow(v: View) {}
        },
      )
    }
  }

  /**
   * Pad a host around the WebView instead of injecting CSS per screen.
   * `position:fixed; inset:0` / `100dvh` overlays then stay inside the
   * already-inset viewport. Consume insets so Chromium 140+ does not also
   * report env(safe-area-inset-*) and double-pad.
   */
  private fun installSafeAreaHost(webView: WebView) {
    if (webView.parent is SafeAreaHost) {
      return
    }
    val parent = webView.parent as? ViewGroup ?: return
    val index = parent.indexOfChild(webView)
    val layoutParams = webView.layoutParams
    parent.removeView(webView)

    val background = themeBackgroundColor()
    val host = SafeAreaHost(this)
    host.addView(
      webView,
      FrameLayout.LayoutParams(
        FrameLayout.LayoutParams.MATCH_PARENT,
        FrameLayout.LayoutParams.MATCH_PARENT,
      ),
    )
    parent.addView(host, index, layoutParams)
    safeAreaHost = host
    applyChromeColor(background)

    ViewCompat.setOnApplyWindowInsetsListener(host) { view, insets ->
      val bars = insets.getInsets(
        WindowInsetsCompat.Type.systemBars() or WindowInsetsCompat.Type.displayCutout(),
      )
      val imeVisible = insets.getInsets(WindowInsetsCompat.Type.ime()).bottom > 0
      view.setPadding(bars.left, bars.top, bars.right, if (imeVisible) 0 else bars.bottom)
      WindowInsetsCompat.CONSUMED
    }
    ViewCompat.requestApplyInsets(host)
  }

  private fun applyChromeColor(color: Int) {
    safeAreaHost?.setBackgroundColor(color)
    appWebView?.setBackgroundColor(color)
    window.decorView.setBackgroundColor(color)
    val lightContent = isLightColor(color)
    WindowInsetsControllerCompat(window, window.decorView).apply {
      isAppearanceLightStatusBars = lightContent
      isAppearanceLightNavigationBars = lightContent
    }
  }

  private fun themeBackgroundColor(): Int {
    val value = TypedValue()
    return if (theme.resolveAttribute(android.R.attr.colorBackground, value, true)) {
      value.data
    } else {
      Color.BLACK
    }
  }

  @Keep
  inner class SafeAreaBridge {
    @JavascriptInterface
    fun setBackground(cssColor: String) {
      val parsed = parseCssColor(cssColor) ?: return
      runOnUiThread { applyChromeColor(parsed) }
    }
  }

  private class SafeAreaHost(context: android.content.Context) : FrameLayout(context)

  companion object {
    private const val BRIDGE_NAME = "LiveAgentSafeArea"
    private const val THEME_SCRIPT =
      "(function(){var last='';function read(){var h=document.documentElement;" +
        "if(!h)return '';var raw=(getComputedStyle(h).getPropertyValue('--background')||'').trim();" +
        "if(raw)return 'hsl('+raw+')';var el=document.body||h;" +
        "return getComputedStyle(el).backgroundColor||'';}function post(){try{var c=read();" +
        "if(!c||c===last||!window.LiveAgentSafeArea)return;last=c;" +
        "LiveAgentSafeArea.setBackground(c);}catch(e){}}function start(){" +
        "post();new MutationObserver(post).observe(document.documentElement," +
        "{attributes:true,attributeFilter:['class','style']});}" +
        "if(document.documentElement)start();" +
        "document.addEventListener('DOMContentLoaded',post);})();"

    private fun isLightColor(color: Int): Boolean {
      val r = Color.red(color)
      val g = Color.green(color)
      val b = Color.blue(color)
      return (0.299 * r + 0.587 * g + 0.114 * b) > 140.0
    }

    private fun parseCssColor(input: String): Int? {
      val s = input.trim()
      if (s.isEmpty()) return null
      if (s.startsWith("#")) {
        return try {
          Color.parseColor(if (s.length == 4) expandShortHex(s) else s)
        } catch (_: IllegalArgumentException) {
          null
        }
      }
      val rgb = Regex(
        """^rgba?\(\s*([\d.]+)\s*[, ]\s*([\d.]+)\s*[, ]\s*([\d.]+)""",
        RegexOption.IGNORE_CASE,
      ).find(s)
      if (rgb != null) {
        return Color.rgb(
          rgb.groupValues[1].toFloat().roundToInt().coerceIn(0, 255),
          rgb.groupValues[2].toFloat().roundToInt().coerceIn(0, 255),
          rgb.groupValues[3].toFloat().roundToInt().coerceIn(0, 255),
        )
      }
      val hsl = Regex(
        """^hsla?\(\s*([\d.]+)[,\s]+([\d.]+)%[,\s]+([\d.]+)%""",
        RegexOption.IGNORE_CASE,
      ).find(s) ?: return null
      return hslToColor(
        hsl.groupValues[1].toFloat(),
        hsl.groupValues[2].toFloat() / 100f,
        hsl.groupValues[3].toFloat() / 100f,
      )
    }

    private fun expandShortHex(short: String): String {
      val c = short.substring(1)
      return "#" + c[0] + c[0] + c[1] + c[1] + c[2] + c[2]
    }

    private fun hslToColor(hue: Float, saturation: Float, lightness: Float): Int {
      val h = ((hue % 360f) + 360f) % 360f
      val s = saturation.coerceIn(0f, 1f)
      val l = lightness.coerceIn(0f, 1f)
      val c = (1f - abs(2f * l - 1f)) * s
      val hp = h / 60f
      val x = c * (1f - abs(hp % 2f - 1f))
      val (r1, g1, b1) = when {
        hp < 1f -> Triple(c, x, 0f)
        hp < 2f -> Triple(x, c, 0f)
        hp < 3f -> Triple(0f, c, x)
        hp < 4f -> Triple(0f, x, c)
        hp < 5f -> Triple(x, 0f, c)
        else -> Triple(c, 0f, x)
      }
      val m = l - c / 2f
      return Color.rgb(
        ((r1 + m) * 255f).roundToInt().coerceIn(0, 255),
        ((g1 + m) * 255f).roundToInt().coerceIn(0, 255),
        ((b1 + m) * 255f).roundToInt().coerceIn(0, 255),
      )
    }
  }
}
