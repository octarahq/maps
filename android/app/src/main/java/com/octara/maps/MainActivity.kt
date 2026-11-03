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
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.BottomSheetScaffold
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.rememberBottomSheetScaffoldState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.navigation.Screen
import com.octara.maps.navigation.rememberRouter
import com.octara.maps.ui.screens.map.StopInfoSheetContent
import com.octara.maps.ui.screens.map.TransitMap
import com.octara.maps.ui.screens.map.TransitMapSheetContent
import com.octara.maps.ui.screens.search.SearchScreen
import com.octara.maps.ui.theme.AppDimensions
import com.octara.maps.utils.LocationHelper
import kotlinx.coroutines.launch
import org.maplibre.android.MapLibre
import org.maplibre.android.geometry.LatLng

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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MapsApp() {
    val router = rememberRouter()
    val context = LocalContext.current
    val coroutineScope = rememberCoroutineScope()
    val locationHelper = remember { LocationHelper(context) }
    val configuration = LocalConfiguration.current
    val density = LocalDensity.current

    var userCoords by remember { mutableStateOf(Pair(48.8566f, 2.3522f)) }

    val permissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestMultiplePermissions()
    ) { permissions ->
        val granted = permissions[Manifest.permission.ACCESS_FINE_LOCATION] == true ||
                      permissions[Manifest.permission.ACCESS_COARSE_LOCATION] == true

        if (granted) {
            coroutineScope.launch {
                locationHelper.getCurrentLocation()?.let { userCoords = it }
            }
        }
    }

    LaunchedEffect(Unit) {
        val hasFine = ContextCompat.checkSelfPermission(
            context, Manifest.permission.ACCESS_FINE_LOCATION
        ) == PackageManager.PERMISSION_GRANTED

        val hasCoarse = ContextCompat.checkSelfPermission(
            context, Manifest.permission.ACCESS_COARSE_LOCATION
        ) == PackageManager.PERMISSION_GRANTED

        if (hasFine || hasCoarse) {
            locationHelper.getCurrentLocation()?.let { userCoords = it }
        } else {
            permissionLauncher.launch(
                arrayOf(
                    Manifest.permission.ACCESS_FINE_LOCATION,
                    Manifest.permission.ACCESS_COARSE_LOCATION
                )
            )
        }
    }

    val currentScreen = router.currentScreen
    val isStopInfo = currentScreen is Screen.StopInfo

    var targetCoords by remember { mutableStateOf<LatLng?>(null) }
    var targetTitle by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(currentScreen) {
        if (currentScreen is Screen.StopInfo) {
            if (currentScreen.lat != null && currentScreen.lon != null) {
                targetCoords = LatLng(currentScreen.lat, currentScreen.lon)
                targetTitle = currentScreen.name
            }
        } else if (currentScreen is Screen.Map) {
            targetCoords = null
            targetTitle = null
        }
    }

    val stopInfoPeekHeight = remember(configuration.screenHeightDp) {
        (configuration.screenHeightDp.dp * 0.42f)
            .coerceIn(AppDimensions.StopInfoSheetPeekHeight, 420.dp)
    }

    val sheetPeekHeight = if (isStopInfo) stopInfoPeekHeight else AppDimensions.BottomSheetPeekHeight
    val sheetPeekHeightPx = with(density) { sheetPeekHeight.roundToPx() }

    val scaffoldState = rememberBottomSheetScaffoldState()

    LaunchedEffect(isStopInfo) {
        try {
            scaffoldState.bottomSheetState.partialExpand()
        } catch (_: Exception) {}
    }

    CompositionLocalProvider(
        LocalRouter provides router
    ) {
        BackHandler(enabled = router.canGoBack) {
            router.goBack()
        }

        Box(modifier = Modifier.fillMaxSize()) {
            BottomSheetScaffold(
                scaffoldState = scaffoldState,
                sheetShape = AppDimensions.BottomSheetShape,
                sheetPeekHeight = sheetPeekHeight,
                sheetShadowElevation = AppDimensions.BottomSheetElevation,
                sheetContent = {
                    when (val screen = router.currentScreen) {
                        is Screen.StopInfo -> {
                            key(screen.stopId) {
                                StopInfoSheetContent(
                                    stopId = screen.stopId,
                                    onStopLoaded = { lat, lon, name ->
                                        if (targetCoords == null) {
                                            targetCoords = LatLng(lat, lon)
                                            targetTitle = name
                                        }
                                    }
                                )
                            }
                        }
                        else -> {
                            TransitMapSheetContent(
                                onSearchClick = { router.navigateTo(Screen.Search) }
                            )
                        }
                    }
                },
                modifier = Modifier.fillMaxSize()
            ) {
                Box(modifier = Modifier.fillMaxSize()) {
                    TransitMap(
                        modifier = Modifier.fillMaxSize(),
                        cameraTarget = targetCoords,
                        cameraZoom = 15.5,
                        bottomPaddingPx = if (isStopInfo) sheetPeekHeightPx else with(density) { AppDimensions.BottomSheetPeekHeight.roundToPx() },
                        markerPosition = targetCoords,
                        markerTitle = targetTitle
                    )

                    AnimatedVisibility(
                        visible = isStopInfo,
                        enter = fadeIn(),
                        exit = fadeOut(),
                        modifier = Modifier
                            .statusBarsPadding()
                            .padding(start = 16.dp, top = 8.dp)
                            .align(Alignment.TopStart)
                    ) {
                        Surface(
                            shape = CircleShape,
                            color = MaterialTheme.colorScheme.surface.copy(alpha = 0.92f),
                            shadowElevation = 4.dp
                        ) {
                            IconButton(
                                onClick = { router.goBack() },
                                modifier = Modifier.size(42.dp)
                            ) {
                                Icon(
                                    imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                                    contentDescription = "Retour",
                                    tint = MaterialTheme.colorScheme.onSurface
                                )
                            }
                        }
                    }
                }
            }

            AnimatedVisibility(
                visible = router.currentScreen is Screen.Search,
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