package com.octara.maps.navigation

import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.SheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.ProvidableCompositionLocal
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.octara.maps.ui.screens.map.StopInfoSheetContent
import com.octara.maps.ui.screens.map.TransitMapSheetContent
import com.octara.maps.ui.screens.search.SearchScreen
import com.octara.maps.ui.theme.AppDimensions
import org.maplibre.android.geometry.LatLng

@OptIn(ExperimentalMaterial3Api::class)
sealed interface Screen {
    val isDismissible: Boolean get() = this !is Map
    val showBackButton: Boolean get() = this !is Map
    val isOverlay: Boolean get() = false

    val mapTarget: LatLng? get() = null
    val mapTitle: String? get() = null
    val cameraZoom: Double get() = 15.5

    fun sheetPeekHeight(screenHeightDp: Dp): Dp = AppDimensions.BottomSheetPeekHeight

    @Composable
    fun SheetContent(
        sheetState: SheetState,
        onClose: () -> Unit,
        onUpdateTarget: (LatLng, String?) -> Unit
    ) {}

    @Composable
    fun OverlayContent(onBack: () -> Unit) {}

    data object Map : Screen {
        override val isDismissible: Boolean = false
        override val showBackButton: Boolean = false

        @Composable
        override fun SheetContent(
            sheetState: SheetState,
            onClose: () -> Unit,
            onUpdateTarget: (LatLng, String?) -> Unit
        ) {
            val router = LocalRouter.current
            TransitMapSheetContent(
                onSearchClick = { router.navigateTo(Search) }
            )
        }
    }

    data object Search : Screen {
        override val isOverlay: Boolean = true

        @Composable
        override fun OverlayContent(onBack: () -> Unit) {
            SearchScreen(onBack = onBack)
        }
    }

    data class StopInfo(
        val stopId: String,
        val lat: Double? = null,
        val lon: Double? = null,
        val name: String? = null
    ) : Screen {
        override val isDismissible: Boolean = true
        override val showBackButton: Boolean = true

        override val mapTarget: LatLng?
            get() = if (lat != null && lon != null) LatLng(lat, lon) else null

        override val mapTitle: String?
            get() = name

        override fun sheetPeekHeight(screenHeightDp: Dp): Dp {
            return (screenHeightDp * 0.42f)
                .coerceIn(AppDimensions.StopInfoSheetPeekHeight, 420.dp)
        }

        @Composable
        override fun SheetContent(
            sheetState: SheetState,
            onClose: () -> Unit,
            onUpdateTarget: (LatLng, String?) -> Unit
        ) {
            StopInfoSheetContent(
                stopId = stopId,
                sheetState = sheetState,
                onClose = onClose,
                onStopLoaded = { stopLat, stopLon, stopName ->
                    onUpdateTarget(LatLng(stopLat, stopLon), stopName)
                }
            )
        }
    }
}

class Router(initialScreen: Screen = Screen.Map) {
    var currentScreen by mutableStateOf(initialScreen)
        private set

    var activeMapScreen by mutableStateOf(
        if (!initialScreen.isOverlay) initialScreen else Screen.Map
    )
        private set

    private val backStack = mutableListOf(initialScreen)

    val canGoBack: Boolean
        get() = backStack.size > 1

    fun navigateTo(screen: Screen, popCurrent: Boolean = false) {
        if (popCurrent && backStack.size > 1) {
            backStack.removeAt(backStack.size - 1)
        }
        if (currentScreen != screen) {
            backStack.add(screen)
            currentScreen = screen
            if (!screen.isOverlay) {
                activeMapScreen = screen
            }
        }
    }

    fun goBack(): Boolean {
        if (canGoBack) {
            backStack.removeAt(backStack.size - 1)
            currentScreen = backStack.last()
            activeMapScreen = backStack.findLast { !it.isOverlay } ?: Screen.Map
            return true
        }
        return false
    }

    fun popToRoot() {
        if (backStack.size > 1) {
            val root = backStack.first()
            backStack.clear()
            backStack.add(root)
            currentScreen = root
            activeMapScreen = backStack.findLast { !it.isOverlay } ?: Screen.Map
        }
    }
}

val LocalRouter: ProvidableCompositionLocal<Router> = compositionLocalOf {
    error("no router provided")
}

@Composable
fun rememberRouter(initialScreen: Screen = Screen.Map): Router {
    return remember { Router(initialScreen) }
}
