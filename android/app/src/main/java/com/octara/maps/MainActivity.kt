package com.octara.maps

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.ContextCompat
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.managers.ActiveNavigationManager
import com.octara.maps.managers.FavoritesStopsManager
import com.octara.maps.navigation.Screen
import com.octara.maps.navigation.rememberRouter
import com.octara.maps.ui.screens.map.TransitMapScreen
import com.octara.maps.ui.theme.OctaraMapsTheme
import com.octara.maps.utils.LocationHelper
import com.octara.maps.utils.LocationManager
import kotlinx.coroutines.launch
import org.maplibre.android.MapLibre

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.auto(
                android.graphics.Color.TRANSPARENT,
                android.graphics.Color.TRANSPARENT
            ),
            navigationBarStyle = SystemBarStyle.auto(
                android.graphics.Color.TRANSPARENT,
                android.graphics.Color.TRANSPARENT
            )
        )
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            window.isNavigationBarContrastEnforced = false
        }
        super.onCreate(savedInstanceState)

        MapLibre.getInstance(this)
        FavoritesStopsManager.initialize(applicationContext)

        setContent {
            OctaraMapsTheme {
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
    val context = LocalContext.current

    val permissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestMultiplePermissions()
    ) { permissions ->
        val granted = permissions[Manifest.permission.ACCESS_FINE_LOCATION] == true ||
                      permissions[Manifest.permission.ACCESS_COARSE_LOCATION] == true

        if (granted) {
            LocationManager.startLocationUpdates(context)
        }
    }

    LaunchedEffect(Unit) {
        val savedNav = ActiveNavigationManager.loadActiveNavigationIfStillValid(context)
        if (savedNav != null && router.currentScreen !is Screen.Navigation) {
            router.navigateTo(Screen.Navigation(trip = savedNav))
        }
        if (LocationManager.hasLocationPermission(context)) {
            LocationManager.startLocationUpdates(context)
        } else {
            permissionLauncher.launch(
                arrayOf(
                    Manifest.permission.ACCESS_FINE_LOCATION,
                    Manifest.permission.ACCESS_COARSE_LOCATION
                )
            )
        }
    }

    CompositionLocalProvider(
        LocalRouter provides router
    ) {
        BackHandler(enabled = router.canGoBack) {
            router.goBack()
        }

        Box(modifier = Modifier.fillMaxSize()) {
            TransitMapScreen()

            AnimatedVisibility(
                visible = router.currentScreen.isOverlay,
                enter = fadeIn() + slideInVertically { it / 6 },
                exit = fadeOut() + slideOutVertically { it / 6 }
            ) {
                router.currentScreen.OverlayContent(
                    onBack = { router.goBack() }
                )
            }
        }
    }
}
