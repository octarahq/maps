package com.octara.maps

import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.navigation.Screen
import com.octara.maps.navigation.rememberRouter
import com.octara.maps.ui.screens.map.TransitMapScreen
import com.octara.maps.ui.screens.search.SearchScreen
import org.maplibre.android.MapLibre

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.light(
                android.graphics.Color.TRANSPARENT,
                android.graphics.Color.TRANSPARENT
            ),
            navigationBarStyle = SystemBarStyle.light(
                android.graphics.Color.TRANSPARENT,
                android.graphics.Color.TRANSPARENT
            )
        )
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            window.isNavigationBarContrastEnforced = false
        }
        super.onCreate(savedInstanceState)

        MapLibre.getInstance(this)

        setContent {
            MaterialTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    MapsApp()
                }
            }
        }
    }
}

@Composable
fun MapsApp() {
    val router = rememberRouter()

    CompositionLocalProvider(LocalRouter provides router) {
        BackHandler(enabled = router.canGoBack) {
            router.goBack()
        }

        Box(modifier = Modifier.fillMaxSize()) {
            TransitMapScreen(
                onSearchClick = { router.navigateTo(Screen.Search) }
            )

            AnimatedVisibility(
                visible = router.currentScreen == Screen.Search,
                enter = fadeIn() + slideInVertically { it / 6 },
                exit = fadeOut() + slideOutVertically { it / 6 }
            ) {
                SearchScreen(
                    onBack = { router.goBack() }
                )
            }
        }
    }
}